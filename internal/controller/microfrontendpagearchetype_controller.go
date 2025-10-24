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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/validate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndPageArchetypeReconciler reconciles a MicroFrontEndPageArchetype object
type MicroFrontEndPageArchetypeReconciler struct {
	MicroFrontEndCommonReconciler
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

	if err := validate.TemplateContent(
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

	if pageArchetype.Spec.DefaultFooterRef != nil {
		var footer kdexv1alpha1.MicroFrontEndPageFooter
		footerName := types.NamespacedName{
			Name:      pageArchetype.Spec.DefaultFooterRef.Name,
			Namespace: pageArchetype.Namespace,
		}

		if err := r.Get(ctx, footerName, &footer); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageArchetype.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageFooter %s not found", footerName.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageArchetype); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageFooter", "name", footerName.Name)
			return ctrl.Result{}, err
		}
	}

	if pageArchetype.Spec.DefaultHeaderRef != nil {
		var header kdexv1alpha1.MicroFrontEndPageHeader
		headerName := types.NamespacedName{
			Name:      pageArchetype.Spec.DefaultHeaderRef.Name,
			Namespace: pageArchetype.Namespace,
		}

		if err := r.Get(ctx, headerName, &header); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageArchetype.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageHeader %s not found", headerName.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageArchetype); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageHeader", "name", headerName.Name)
			return ctrl.Result{}, err
		}
	}

	if pageArchetype.Spec.DefaultMainNavigationRef != nil {
		navigation, response, err := r.GetNavigation(
			ctx, log, *pageArchetype.Spec.DefaultMainNavigationRef, &pageArchetype.Status.Conditions, &pageArchetype)

		if navigation == nil {
			return response, err
		}
	}

	if pageArchetype.Spec.ExtraNavigations != nil {
		for _, navigationRef := range *pageArchetype.Spec.ExtraNavigations {
			navigation, response, err := r.GetNavigation(
				ctx, log, navigationRef, &pageArchetype.Status.Conditions, &pageArchetype)

			if navigation == nil {
				return response, err
			}
		}
	}

	if pageArchetype.Spec.OverrideStylesheetRef != nil {
		var stylesheet kdexv1alpha1.MicroFrontEndStylesheet
		stylesheetName := types.NamespacedName{
			Name:      pageArchetype.Spec.OverrideStylesheetRef.Name,
			Namespace: pageArchetype.Namespace,
		}

		if err := r.Get(ctx, stylesheetName, &stylesheet); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageArchetype.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndStylesheet %s not found", stylesheetName.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageArchetype); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndStylesheet", "name", stylesheetName.Name)
			return ctrl.Result{}, err
		}
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
