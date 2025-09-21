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
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"kdex.dev/app-server/internal/customelement"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MicroFrontEndPageBindingReconciler reconciles a MicroFrontEndPageBinding object
type MicroFrontEndPageBindingReconciler struct {
	MicroFrontEndCommonReconciler
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendrenderpages,verbs=get;list;watch;create;update;patch;delete

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
			log.Error(err, "referenced MicroFrontEndPageArchetype %s not found", pageBinding.Spec.PageArchetypeRef.Name)
			apimeta.SetStatusCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndPageArchetype %s not found", pageBinding.Spec.PageArchetypeRef.Name),
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

	if !apimeta.IsStatusConditionTrue(pageArchetype.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady)) {
		log.Error(fmt.Errorf("referenced MicroFrontEndPageArchetype %s is not ready", pageBinding.Spec.PageArchetypeRef.Name), "")
		apimeta.SetStatusCondition(
			&pageBinding.Status.Conditions,
			*kdexv1alpha1.NewCondition(
				kdexv1alpha1.ConditionTypeReady,
				metav1.ConditionFalse,
				kdexv1alpha1.ConditionReasonReconcileError,
				fmt.Sprintf("referenced MicroFrontEndPageArchetype %s is not ready", pageBinding.Spec.PageArchetypeRef.Name),
			),
		)
		if err := r.Status().Update(ctx, &pageBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	contents, response, err := r.contents(ctx, log, &pageBinding)
	if err != nil {
		return response, err
	}

	navigations, response, err := r.navigations(ctx, log, &pageBinding, &pageArchetype)
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
				log.Error(err, "referenced MicroFrontEndPageHeader %s not found", headerRef.Name)
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageHeader %s not found", headerRef.Name),
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
				log.Error(err, "referenced MicroFrontEndPageFooter %s not found", footerRef.Name)
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageFooter %s not found", footerRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageBinding); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageFooter %s", footerRef.Name)
			return ctrl.Result{}, err
		}
	}

	renderPage := &kdexv1alpha1.MicroFrontEndRenderPage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pageBinding.Name,
			Namespace: pageBinding.Namespace,
		},
		Spec: kdexv1alpha1.MicroFrontEndRenderPageSpec{
			NavigationHints: pageBinding.Spec.NavigationHints,
			Path:            pageBinding.Spec.Path,
			PageComponents: kdexv1alpha1.PageComponents{
				Contents:        contents,
				Footer:          footer.Spec.Content,
				Header:          header.Spec.Content,
				Navigations:     navigations,
				PrimaryTemplate: pageArchetype.Spec.Content,
				Title:           pageBinding.Spec.Label,
			},
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, renderPage, func() error {
		return ctrl.SetControllerReference(&pageBinding, renderPage, r.Scheme)
	}); err != nil {
		log.Error(err, "unable to create or update MicroFrontEndRenderPage")
		return ctrl.Result{}, err
	}

	log.Info("reconciled MicroFrontEndPageBinding", "pageBinding", pageBinding, "renderPage", renderPage)

	apimeta.SetStatusCondition(
		&pageBinding.Status.Conditions,
		*kdexv1alpha1.NewCondition(
			kdexv1alpha1.ConditionTypeReady,
			metav1.ConditionTrue,
			kdexv1alpha1.ConditionReasonReconcileSuccess,
			"all references resolved successfully",
		),
	)
	if err := r.Status().Update(ctx, &pageBinding); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroFrontEndPageBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.MicroFrontEndPageBinding{}).
		Owns(&kdexv1alpha1.MicroFrontEndRenderPage{}).
		Watches(
			&kdexv1alpha1.MicroFrontEndApp{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForApp)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageArchetype{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageArchetype)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageFooter{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageFooter)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageHeader{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageHeader)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageNavigations)).
		Named("microfrontendpagebinding").
		Complete(r)
}

func (r *MicroFrontEndPageBindingReconciler) contents(
	ctx context.Context,
	log logr.Logger,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
) (map[string]string, ctrl.Result, error) {
	contents := make(map[string]string)

	for _, contentEntry := range pageBinding.Spec.ContentEntries {
		appRef := contentEntry.AppRef
		if appRef == nil {
			contents[contentEntry.Slot] = contentEntry.RawHTML

			continue
		}

		appName := types.NamespacedName{
			Name:      appRef.Name,
			Namespace: pageBinding.Namespace,
		}
		var app kdexv1alpha1.MicroFrontEndApp
		if err := r.Get(ctx, appName, &app); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "referenced MicroFrontEndApp %s not found", appRef.Name)
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndApp %s not found", appRef.Name),
					),
				)
				if err := r.Status().Update(ctx, pageBinding); err != nil {
					return nil, ctrl.Result{}, err
				}

				return nil, ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndApp %s", appRef.Name)
			return nil, ctrl.Result{}, err
		}

		if !apimeta.IsStatusConditionTrue(app.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady)) {
			log.Error(fmt.Errorf("referenced MicroFrontEndApp %s is not ready", appRef.Name), "")
			apimeta.SetStatusCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndApp %s is not ready", appRef.Name),
				),
			)
			if err := r.Status().Update(ctx, pageBinding); err != nil {
				return nil, ctrl.Result{}, err
			}

			return nil, ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}

		contents[contentEntry.Slot] = customelement.ForApp(app, contentEntry, *pageBinding)
	}

	return contents, ctrl.Result{}, nil
}

func (r *MicroFrontEndPageBindingReconciler) navigations(
	ctx context.Context,
	log logr.Logger,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	pageArchetype *kdexv1alpha1.MicroFrontEndPageArchetype,
) (map[string]string, ctrl.Result, error) {
	navigations := make(map[string]string)

	navigationRef := pageBinding.Spec.OverrideMainNavigationRef
	if navigationRef == nil {
		navigationRef = pageArchetype.Spec.DefaultMainNavigationRef
	}
	if navigationRef != nil {
		navigation, response, err := r.GetNavigation(
			ctx, log, *navigationRef, ClientObjectWithConditions{
				Object:     pageBinding,
				Conditions: pageBinding.Status.Conditions,
			})

		if navigation == nil {
			return nil, response, err
		}

		navigations["main"] = navigation.Spec.Content
	}

	if pageArchetype.Spec.ExtraNavigations == nil {
		return navigations, ctrl.Result{}, nil
	}

	for navigationName, navigationRef := range *pageArchetype.Spec.ExtraNavigations {
		navigation, response, err := r.GetNavigation(
			ctx, log, navigationRef, ClientObjectWithConditions{
				Object:     pageBinding,
				Conditions: pageBinding.Status.Conditions,
			})

		if navigation == nil {
			return nil, response, err
		}

		navigations[navigationName] = navigation.Spec.Content
	}

	return navigations, ctrl.Result{}, nil
}
