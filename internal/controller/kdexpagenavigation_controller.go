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

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexPageNavigationReconciler reconciles a KDexPageNavigation object
type KDexPageNavigationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations/finalizers,verbs=update

func (r *KDexPageNavigationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageNavigation kdexv1alpha1.KDexPageNavigation
	if err := r.Get(ctx, req.NamespacedName, &pageNavigation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := render.ValidateContent(
		pageNavigation.Name, pageNavigation.Spec.Content,
	); err != nil {
		apimeta.SetStatusCondition(
			&pageNavigation.Status.Conditions,
			*kdexv1alpha1.NewCondition(
				kdexv1alpha1.ConditionTypeReady,
				metav1.ConditionFalse,
				kdexv1alpha1.ConditionReasonReconcileError,
				err.Error(),
			),
		)
		if err := r.Status().Update(ctx, &pageNavigation); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexPageNavigation")

	apimeta.SetStatusCondition(
		&pageNavigation.Status.Conditions,
		*kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionTrue,
			kdexv1alpha1.ConditionReasonReconcileSuccess,
			"content template is valid",
		),
	)
	if err := r.Status().Update(ctx, &pageNavigation); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageNavigationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageNavigation{}).
		Named("kdexpagenavigation").
		Complete(r)
}
