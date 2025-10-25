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

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndPageArchetypeReconciler reconciles a MicroFrontEndPageArchetype object
type MicroFrontEndPageArchetypeReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	RequeueDelay time.Duration
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagefooters,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpageheaders,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagenavigations,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendstylesheets,verbs=get;list;watch

func (r *MicroFrontEndPageArchetypeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageArchetype kdexv1alpha1.MicroFrontEndPageArchetype
	if err := r.Get(ctx, req.NamespacedName, &pageArchetype); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := render.ValidateContent(
		pageArchetype.Name, pageArchetype.Spec.Content,
	); err != nil {
		apimeta.SetStatusCondition(
			&pageArchetype.Status.Conditions,
			*kdexv1alpha1.NewCondition(
				kdexv1alpha1.ConditionTypeReady,
				metav1.ConditionFalse,
				kdexv1alpha1.ConditionReasonReconcileError,
				err.Error(),
			),
		)
		if err := r.Status().Update(ctx, &pageArchetype); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	_, shouldReturn, r1, err := resolvePageFooter(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.DefaultFooterRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, r1, err = resolvePageHeader(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.DefaultHeaderRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, response, err := resolvePageNavigations(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.DefaultMainNavigationRef, pageArchetype.Spec.ExtraNavigations, r.RequeueDelay)
	if err != nil {
		return response, err
	}

	_, shouldReturn, r1, err = resolveStylesheet(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.OverrideStylesheetRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	log.Info("reconciled MicroFrontEndPageArchetype")

	apimeta.SetStatusCondition(
		&pageArchetype.Status.Conditions,
		*kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionTrue,
			kdexv1alpha1.ConditionReasonReconcileSuccess,
			"all references resolved successfully",
		),
	)
	if err := r.Status().Update(ctx, &pageArchetype); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroFrontEndPageArchetypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.MicroFrontEndPageArchetype{}).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageFooter{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageFooter)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageHeader{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageHeader)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageNavigations)).
		Watches(
			&kdexv1alpha1.MicroFrontEndStylesheet{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForStylesheet)).
		Named("microfrontendpagearchetype").
		Complete(r)
}
