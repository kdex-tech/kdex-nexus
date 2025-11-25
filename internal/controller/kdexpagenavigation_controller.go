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
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KDexPageNavigationReconciler reconciles a KDexPageNavigation object
type KDexPageNavigationReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,                  verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations/status,           verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations/finalizers,       verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagenavigations,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagenavigations/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagenavigations/finalizers,verbs=update

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

func (r *KDexPageNavigationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexPageNavigationSpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterPageNavigation kdexv1alpha1.KDexClusterPageNavigation
		if err := r.Get(ctx, req.NamespacedName, &clusterPageNavigation); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterPageNavigation.Status
		spec = clusterPageNavigation.Spec
		o = &clusterPageNavigation
	} else {
		var pageNavigation kdexv1alpha1.KDexPageNavigation
		if err := r.Get(ctx, req.NamespacedName, &pageNavigation); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &pageNavigation.Status
		spec = pageNavigation.Spec
		o = &pageNavigation
	}

	// Defer status update
	defer func() {
		status.ObservedGeneration = om.Generation
		if updateErr := r.Status().Update(ctx, o); updateErr != nil {
			if res == (ctrl.Result{}) {
				err = updateErr
			}
		}
	}()

	kdexv1alpha1.SetConditions(
		&status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)

	_, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if err := render.ValidateContent(
		o.GetName(), spec.Content,
	); err != nil {
		kdexv1alpha1.SetConditions(
			&status.Conditions,
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
		&status.Conditions,
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
			&kdexv1alpha1.KDexClusterPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: o.GetName()}}}
			}),
		).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageNavigationsForScriptLibrary),
		).
		Named("kdexpagenavigation").
		Complete(r)
}
