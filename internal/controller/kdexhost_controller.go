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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const hostFinalizerName = "kdex.dev/kdex-nexus-host-finalizer"

// KDexHostReconciler reconciles a KDexHost object
type KDexHostReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes,verbs=get;list;watch

func (r *KDexHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var host kdexv1alpha1.KDexHost
	if err := r.Get(ctx, req.NamespacedName, &host); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if host.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&host, hostFinalizerName) {
			controllerutil.AddFinalizer(&host, hostFinalizerName)
			if err := r.Update(ctx, &host); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(&host, hostFinalizerName) {

			// TODO delete the things...

			controllerutil.RemoveFinalizer(&host, hostFinalizerName)
			if err := r.Update(ctx, &host); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	kdexv1alpha1.SetConditions(
		&host.Status.Conditions,
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
	if err := r.Status().Update(ctx, &host); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		host.Status.ObservedGeneration = host.Generation
		if err := r.Status().Update(ctx, &host); err != nil {
			log.Error(err, "failed to update host status")
		}
	}()

	_, shouldReturn, r1, err := resolveTheme(ctx, r.Client, &host, &host.Status.Conditions, host.Spec.DefaultThemeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, r1, err = resolveScriptLibrary(ctx, r.Client, &host, &host.Status.Conditions, host.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if err := executeFocusedHostControllerSetup(ctx, &host); err != nil {
		kdexv1alpha1.SetConditions(
			&host.Status.Conditions,
			kdexv1alpha1.ConditionArgs{
				Degraded: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionTrue,
					Reason:  "FocusedHostControllerSetupFailed",
					Message: err.Error(),
				},
				Progressing: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "FocusedHostControllerSetupFailed",
					Message: err.Error(),
				},
				Ready: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "FocusedHostControllerSetupFailed",
					Message: err.Error(),
				},
			},
		)
		if err := r.Status().Update(ctx, &host); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	kdexv1alpha1.SetConditions(
		&host.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Stage 1 Reconciliation successful",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Stage 1 Reconciliation successful",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Stage 1 Reconciliation successful",
			},
		},
	)
	if err := r.Status().Update(ctx, &host); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexHost stage1")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexHost{}).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findHostsForScriptLibrary),
		).
		Watches(
			&kdexv1alpha1.KDexTheme{},
			handler.EnqueueRequestsFromMapFunc(r.findHostsForTheme)).
		Named("kdexhost-stage1").
		Complete(r)
}

func executeFocusedHostControllerSetup(ctx context.Context, kDexHost *kdexv1alpha1.KDexHost) error {
	return nil
}
