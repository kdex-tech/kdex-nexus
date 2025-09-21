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
	"errors"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndAppReconciler reconciles a MicroFrontEndApp object
type MicroFrontEndAppReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *MicroFrontEndAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var app kdexv1alpha1.MicroFrontEndApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		log.Error(err, "unable to fetch MicroFrontEndApp")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate the source of the app.
	validated, err := r.validateSource(ctx, &app)
	if err != nil {
		log.Error(err, "source validation failed")
		apimeta.SetStatusCondition(&app.Status.Conditions, *kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionFalse,
			"SourceValidationFailed",
			err.Error(),
		))
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if !validated {
		if apimeta.IsStatusConditionFalse(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String()) {
			condition := apimeta.FindStatusCondition(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String())
			if condition.Reason == "SourceValidationFailed" {
				log.Info("reconcile failed due to failed validation", "app", app)
				return ctrl.Result{}, nil
			}
		}

		log.Info("source validation not complete")
		apimeta.SetStatusCondition(&app.Status.Conditions, *kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionFalse,
			"SourceValidationInProgress",
			"Source validation is in progress",
		))
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
	}

	log.Info("reconciled MicroFrontEndApp", "app", app)

	return ctrl.Result{}, nil
}

// validateSource fetches, extracts, and validates the source of the app.
// This is a placeholder for the actual implementation.
func (r *MicroFrontEndAppReconciler) validateSource(
	ctx context.Context, app *kdexv1alpha1.MicroFrontEndApp,
) (bool, error) {
	log := logf.FromContext(ctx)

	if apimeta.IsStatusConditionTrue(app.Status.Conditions, kdexv1alpha1.ConditionTypeReady.String()) {
		return true, nil
	}

	go func() {
		log.Info("validating source for MicroFrontEndApp", "app", app.Name)

		// The MicroFrontEndApp spec is expected to have a `source` field.
		// For example:
		// spec:
		//   source:
		//     git:
		//       url: "https://github.com/example/repo.git"
		//       revision: "main"

		// Create a Kubernetes job to:
		// 1. Fetch the source
		// This would involve either fetching the source archive or cloning the git repository specified in app.Spec.Source.URL
		// using native command line tools
		log.Info("fetching source from git repository")

		// 2. Extract and validate the contents
		// This would involve inspecting the fetched source code for required files,
		// configuration, etc.
		// For example, check for package.json, Dockerfile, etc.
		log.Info("validating source contents")

		// 3. Potentially build and deploy
		// This could involve building a container image and creating a Deployment.
		log.Info("building and deploying source")
		// For example, run docker build and create a Kubernetes Deployment.

		err := errors.New("source validation failed")

		if err != nil {
			apimeta.SetStatusCondition(
				&app.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					"SourceValidationFailed",
					err.Error(),
				),
			)
		} else {
			apimeta.SetStatusCondition(
				&app.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionTrue,
					kdexv1alpha1.ConditionReasonReconcileSuccess,
					"all references resolved successfully",
				),
			)
		}

		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "failed to update app status")
		}
	}()

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroFrontEndAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.MicroFrontEndApp{}).
		Named("microfrontendapp").
		Complete(r)
}
