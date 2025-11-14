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
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexPageNavigationReconciler reconciles a KDexPageNavigation object
type KDexPageNavigationReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

func (r *KDexPageNavigationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageNavigation kdexv1alpha1.KDexPageNavigation
	if err := r.Get(ctx, req.NamespacedName, &pageNavigation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	kdexv1alpha1.SetConditions(
		&pageNavigation.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)

	// Defer status update
	defer func() {
		pageNavigation.Status.ObservedGeneration = pageNavigation.Generation
		if err := r.Status().Update(ctx, &pageNavigation); err != nil {
			log.Info("failed to update status", "err", err)
		}
	}()

	_, shouldReturn, r1, err := ResolveScriptLibrary(ctx, r.Client, &pageNavigation, &pageNavigation.Status.Conditions, pageNavigation.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if err := render.ValidateContent(
		pageNavigation.Name, pageNavigation.Spec.Content,
	); err != nil {
		kdexv1alpha1.SetConditions(
			&pageNavigation.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		return ctrl.Result{}, err
	}

	kdexv1alpha1.SetConditions(
		&pageNavigation.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionFalse,
			Ready:       metav1.ConditionTrue,
		},
		kdexv1alpha1.ConditionReasonReconcileSuccess,
		"Reconciliation successful",
	)

	log.Info("reconciled KDexPageNavigation")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageNavigationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageNavigation{}).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageNavigationsForScriptLibrary),
		).
		Named("kdexpagenavigation").
		Complete(r)
}
