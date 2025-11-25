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

// KDexPageFooterReconciler reconciles a KDexPageFooter object
type KDexPageFooterReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters,                  verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters/status,           verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters/finalizers,       verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagefooters,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagefooters/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagefooters/finalizers,verbs=update

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

func (r *KDexPageFooterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexPageFooterSpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterPageFooter kdexv1alpha1.KDexClusterPageFooter
		if err := r.Get(ctx, req.NamespacedName, &clusterPageFooter); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterPageFooter.Status
		spec = clusterPageFooter.Spec
		om = clusterPageFooter.ObjectMeta
		o = &clusterPageFooter
	} else {
		var pageFooter kdexv1alpha1.KDexPageFooter
		if err := r.Get(ctx, req.NamespacedName, &pageFooter); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &pageFooter.Status
		spec = pageFooter.Spec
		om = pageFooter.ObjectMeta
		o = &pageFooter
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

	log.Info("reconciled KDexPageFooter")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageFooterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageFooter{}).
		Watches(
			&kdexv1alpha1.KDexClusterPageFooter{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: o.GetName()}}}
			}),
		).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageFootersForScriptLibrary),
		).
		Named("kdexpagefooter").
		Complete(r)
}
