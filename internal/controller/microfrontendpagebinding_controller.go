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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndPageBindingReconciler reconciles a MicroFrontEndPageBinding object
type MicroFrontEndPageBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MicroFrontEndPageBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *MicroFrontEndPageBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageBinding kdexv1alpha1.MicroFrontEndPageBinding
	if err := r.Get(ctx, req.NamespacedName, &pageBinding); err != nil {
		log.Error(err, "unable to fetch MicroFrontEndPageBinding")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var pageArchetype kdexv1alpha1.MicroFrontEndPageArchetype
	pageArchetypeName := types.NamespacedName{
		Name:      pageBinding.Spec.PageArchetypeRef.Name,
		Namespace: pageBinding.Namespace,
	}
	if err := r.Get(ctx, pageArchetypeName, &pageArchetype); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "Referenced MicroFrontEndPageArchetype %s not found", pageBinding.Spec.PageArchetypeRef.Name)
			kdexv1alpha1.SetCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypePageArchetypeNotFound,
					metav1.ConditionTrue,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("Referenced MicroFrontEndPageArchetype %s not found", pageBinding.Spec.PageArchetypeRef.Name),
				),
			)
			if err := r.Status().Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}

		log.Error(err, "unable to fetch MicroFrontEndPageArchetype %s", pageBinding.Spec.PageArchetypeRef.Name)
		return ctrl.Result{}, err
	}

	apps, response, err := r.apps(ctx, log, pageBinding)
	if err != nil {
		return response, err
	}

	navigations, response, err := r.navigations(ctx, log, pageBinding, pageArchetype)
	if err != nil {
		return response, err
	}

	var header kdexv1alpha1.MicroFrontEndPageHeader
	headerRef := pageBinding.Spec.OverrideHeaderRef
	if headerRef == nil {
		headerRef = pageArchetype.Spec.DefaultHeaderRef
	}
	if headerRef != nil {
		headerName := types.NamespacedName{
			Name:      headerRef.Name,
			Namespace: pageBinding.Namespace,
		}

		if err := r.Get(ctx, headerName, &header); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "Referenced MicroFrontEndPageHeader %s not found", headerRef.Name)
				kdexv1alpha1.SetCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeHeaderNotFound,
						metav1.ConditionTrue,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("Referenced MicroFrontEndPageHeader %s not found", headerRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageBinding); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageHeader %s", headerRef.Name)
			return ctrl.Result{}, err
		}
	}

	var footer kdexv1alpha1.MicroFrontEndPageFooter
	footerRef := pageBinding.Spec.OverrideFooterRef
	if footerRef == nil {
		footerRef = pageArchetype.Spec.DefaultFooterRef
	}
	if footerRef != nil {
		footerName := types.NamespacedName{
			Name:      footerRef.Name,
			Namespace: pageBinding.Namespace,
		}

		if err := r.Get(ctx, footerName, &footer); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "Referenced MicroFrontEndPageFooter %s not found", headerRef.Name)
				kdexv1alpha1.SetCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeFooterNotFound,
						metav1.ConditionTrue,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("Referenced MicroFrontEndPageFooter %s not found", headerRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageBinding); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageFooter %s", headerRef.Name)
			return ctrl.Result{}, err
		}
	}

	log.Info("Reconciled MicroFrontEndPageBinding", pageBinding, pageArchetype, apps, navigations, header, footer)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroFrontEndPageBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		Named("microfrontendpagebinding").
		Complete(r)
}

func (r *MicroFrontEndPageBindingReconciler) apps(
	ctx context.Context,
	log logr.Logger,
	pageBinding kdexv1alpha1.MicroFrontEndPageBinding,
) (map[string]kdexv1alpha1.MicroFrontEndApp, ctrl.Result, error) {
	apps := make(map[string]kdexv1alpha1.MicroFrontEndApp)

	for _, contentEntry := range pageBinding.Spec.ContentEntries {
		appRef := contentEntry.AppRef
		if appRef == nil {
			continue
		}

		appName := types.NamespacedName{
			Name:      appRef.Name,
			Namespace: pageBinding.Namespace,
		}
		var app kdexv1alpha1.MicroFrontEndApp
		if err := r.Get(ctx, appName, &app); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "Referenced MicroFrontEndApp %s not found", appRef.Name)
				kdexv1alpha1.SetCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeAppNotFound,
						metav1.ConditionTrue,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("Referenced MicroFrontEndApp %s not found", appRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageBinding); err != nil {
					return nil, ctrl.Result{}, err
				}

				return nil, ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndApp %s", appRef.Name)
			return nil, ctrl.Result{}, err
		}

		apps[appRef.Name] = app
	}

	return apps, ctrl.Result{}, nil
}

func (r *MicroFrontEndPageBindingReconciler) navigations(
	ctx context.Context,
	log logr.Logger,
	pageBinding kdexv1alpha1.MicroFrontEndPageBinding,
	pageArchetype kdexv1alpha1.MicroFrontEndPageArchetype,
) (map[string]kdexv1alpha1.MicroFrontEndPageNavigation, ctrl.Result, error) {
	navigations := make(map[string]kdexv1alpha1.MicroFrontEndPageNavigation)

	navigationRef := pageBinding.Spec.OverrideMainNavigationRef
	if navigationRef == nil {
		navigationRef = pageArchetype.Spec.DefaultMainNavigationRef
	}
	if navigationRef != nil {
		navigation, response, err := r.navigation(
			ctx, log, navigationRef, pageBinding)

		if navigation == nil {
			return nil, response, err
		}

		navigations["main"] = *navigation
	}

	for navigationName, navigationRef := range pageArchetype.Spec.ExtraNavigations {
		navigation, response, err := r.navigation(
			ctx, log, &navigationRef, pageBinding)

		if navigation == nil {
			return nil, response, err
		}

		navigations[navigationName] = *navigation
	}

	return navigations, ctrl.Result{}, nil
}

func (r *MicroFrontEndPageBindingReconciler) navigation(
	ctx context.Context,
	log logr.Logger,
	navigationRef *corev1.LocalObjectReference,
	pageBinding kdexv1alpha1.MicroFrontEndPageBinding,
) (*kdexv1alpha1.MicroFrontEndPageNavigation, ctrl.Result, error) {
	var navigation kdexv1alpha1.MicroFrontEndPageNavigation
	navigationName := types.NamespacedName{
		Name:      navigationRef.Name,
		Namespace: pageBinding.Namespace,
	}
	if err := r.Get(ctx, navigationName, &navigation); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "Referenced MicroFrontEndPageNavigation %s not found", navigationRef.Name)
			kdexv1alpha1.SetCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeNavigationNotFound,
					metav1.ConditionTrue,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("Referenced MicroFrontEndPageNavigation %s not found", navigationRef.Name),
				),
			)
			if err := r.Status().Update(ctx, &pageBinding); err != nil {
				return nil, ctrl.Result{}, err
			}

			return nil, ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}

		log.Error(err, "unable to fetch MicroFrontEndPageNavigation %s", navigationRef.Name)
		return nil, ctrl.Result{}, err
	}

	return &navigation, ctrl.Result{}, nil
}
