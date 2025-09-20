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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndPageArchetypeReconciler reconciles a MicroFrontEndPageArchetype object
type MicroFrontEndPageArchetypeReconciler struct {
	MicroFrontEndCommonReconciler
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MicroFrontEndPageArchetype object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *MicroFrontEndPageArchetypeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageArchetype kdexv1alpha1.MicroFrontEndPageArchetype
	if err := r.Get(ctx, req.NamespacedName, &pageArchetype); err != nil {
		log.Error(err, "unable to fetch MicroFrontEndPageArchetype")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pageArchetype.Spec.DefaultFooterRef != nil {
		var footer kdexv1alpha1.MicroFrontEndPageFooter
		footerName := types.NamespacedName{
			Name:      pageArchetype.Spec.DefaultFooterRef.Name,
			Namespace: pageArchetype.Namespace,
		}

		if err := r.Get(ctx, footerName, &footer); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "referenced MicroFrontEndPageFooter %s not found", pageArchetype.Spec.DefaultFooterRef.Name)
				apimeta.SetStatusCondition(
					&pageArchetype.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageFooter %s not found", pageArchetype.Spec.DefaultFooterRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageArchetype); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageFooter %s", pageArchetype.Spec.DefaultFooterRef.Name)
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
				log.Error(err, "referenced MicroFrontEndPageHeader %s not found", pageArchetype.Spec.DefaultHeaderRef.Name)
				apimeta.SetStatusCondition(
					&pageArchetype.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageHeader %s not found", pageArchetype.Spec.DefaultHeaderRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageArchetype); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageHeader %s", pageArchetype.Spec.DefaultHeaderRef.Name)
			return ctrl.Result{}, err
		}
	}

	if pageArchetype.Spec.DefaultMainNavigationRef != nil {
		navigation, response, err := r.GetNavigation(
			ctx, log, *pageArchetype.Spec.DefaultMainNavigationRef, ClientObjectWithConditions{
				Object:     &pageArchetype,
				Conditions: pageArchetype.Status.Conditions,
			})

		if navigation == nil {
			return response, err
		}
	}

	if pageArchetype.Spec.ExtraNavigations != nil {
		for _, navigationRef := range *pageArchetype.Spec.ExtraNavigations {
			navigation, response, err := r.GetNavigation(
				ctx, log, navigationRef, ClientObjectWithConditions{
					Object:     &pageArchetype,
					Conditions: pageArchetype.Status.Conditions,
				})

			if navigation == nil {
				return response, err
			}
		}
	}

	log.Info("reconciled MicroFrontEndPageArchetype", "pageArchetype", pageArchetype)

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
		Named("microfrontendpagearchetype").
		Complete(r)
}
