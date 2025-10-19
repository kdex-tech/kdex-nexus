/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/npm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MicroFrontEndAppReconciler reconciles a MicroFrontEndApp object
type MicroFrontEndAppReconciler struct {
	client.Client
	RegistryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) npm.Registry
	RequeueDelay    time.Duration
	Scheme          *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps/finalizers,verbs=update

func (r *MicroFrontEndAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var app kdexv1alpha1.MicroFrontEndApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	secret := corev1.Secret{}
	if app.Spec.PackageReference.SecretRef != nil {
		secretName := types.NamespacedName{
			Name:      app.Spec.PackageReference.SecretRef.Name,
			Namespace: app.Namespace,
		}
		if err := r.Get(ctx, secretName, &secret); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "referenced Secret not found", "name", app.Spec.PackageReference.SecretRef.Name)
				apimeta.SetStatusCondition(
					&app.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced Secret %s not found", app.Spec.PackageReference.SecretRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &app); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}
		}
	}

	if err := r.validatePackageReference(ctx, &app, &secret); err != nil {
		if apimeta.IsStatusConditionFalse(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String()) {
			condition := apimeta.FindStatusCondition(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String())
			if condition.Reason == "PackageValidationFailed" {
				log.Info("reconcile failed due to failed validation", "app", app)
				return ctrl.Result{}, err
			}
		}
	}

	log.Info("reconciled MicroFrontEndApp")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroFrontEndAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.MicroFrontEndApp{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findAppsForSecret),
		).
		Named("microfrontendapp").
		Complete(r)
}

func (r *MicroFrontEndAppReconciler) findAppsForSecret(
	ctx context.Context,
	secret client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	if _, ok := secret.GetAnnotations()["kdex.dev/npm-server-address"]; !ok {
		return []reconcile.Request{}
	}

	var appList kdexv1alpha1.MicroFrontEndAppList
	if err := r.List(ctx, &appList, &client.ListOptions{
		Namespace: secret.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndApps for secret %s", secret.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(appList.Items))
	for _, app := range appList.Items {
		if app.Spec.PackageReference.SecretRef.Name == secret.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
		}
	}
	return requests
}

// validatePackageReference fetches, extracts, and validates the NPM package reference that contains the App.
// This is a placeholder for the actual implementation.
func (r *MicroFrontEndAppReconciler) validatePackageReference(
	ctx context.Context, app *kdexv1alpha1.MicroFrontEndApp, secret *corev1.Secret,
) error {
	log := logf.FromContext(ctx)

	if apimeta.IsStatusConditionTrue(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String()) {
		return nil
	}

	log.Info("validating package reference for MicroFrontEndApp", "app", app.Name)

	if !strings.HasPrefix(app.Spec.PackageReference.Name, "@") || !strings.Contains(app.Spec.PackageReference.Name, "/") {
		apimeta.SetStatusCondition(&app.Status.Conditions, *kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionFalse,
			"PackageValidationFailed",
			fmt.Sprintf("invalid package name, must be scoped with @scope/name: %s", app.Spec.PackageReference.Name),
		))
		if err := r.Status().Update(ctx, app); err != nil {
			return err
		}
		return fmt.Errorf("invalid package name, must be scoped with @scope/name: %s", app.Spec.PackageReference.Name)
	}

	registry := r.RegistryFactory(secret, log.Error)

	var condition metav1.Condition

	if err := registry.ValidatePackage(
		app.Spec.PackageReference.Name,
		app.Spec.PackageReference.Version,
	); err != nil {
		condition = *kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionFalse,
			"PackageValidationFailed",
			err.Error(),
		)
	} else {
		condition = *kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionTrue,
			kdexv1alpha1.ConditionReasonReconcileSuccess,
			"all references resolved successfully",
		)
	}

	appName := types.NamespacedName{
		Name:      app.Name,
		Namespace: app.Namespace,
	}

	if err := r.Get(ctx, appName, app); err != nil {
		return err
	}
	apimeta.SetStatusCondition(
		&app.Status.Conditions,
		condition,
	)
	return r.Status().Update(ctx, app)
}
