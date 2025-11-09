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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/npm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexAppReconciler reconciles a KDexApp object
type KDexAppReconciler struct {
	client.Client
	RegistryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) npm.Registry
	RequeueDelay    time.Duration
	Scheme          *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps/finalizers,verbs=update

func (r *KDexAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var app kdexv1alpha1.KDexApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	kdexv1alpha1.SetConditions(
		&app.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionUnknown,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
		},
	)
	if err := r.Status().Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		app.Status.ObservedGeneration = app.Generation
		if err := r.Status().Update(ctx, &app); err != nil {
			log.Error(err, "failed to update app status")
		}
	}()

	secret, shouldReturn, r1, err := resolveSecret(ctx, r.Client, &app, &app.Status.Conditions, app.Spec.PackageReference.SecretRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if err := validatePackageReference(ctx, &app.Spec.PackageReference, secret, r.RegistryFactory); err != nil {
		kdexv1alpha1.SetConditions(
			&app.Status.Conditions,
			kdexv1alpha1.ConditionArgs{
				Degraded: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionTrue,
					Reason:  "PackageValidationFailed",
					Message: err.Error(),
				},
				Progressing: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "PackageValidationFailed",
					Message: err.Error(),
				},
				Ready: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "PackageValidationFailed",
					Message: err.Error(),
				},
			},
		)
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	kdexv1alpha1.SetConditions(
		&app.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
		},
	)
	if err := r.Status().Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexApp")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexApp{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findAppsForSecret),
		).
		Named("kdexapp").
		Complete(r)
}
