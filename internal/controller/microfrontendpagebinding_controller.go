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
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/customelement"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const pageBindingFinalizerName = "kdex.dev/kdex-nexus-page-binding-finalizer"

// MicroFrontEndPageBindingReconciler reconciles a MicroFrontEndPageBinding object
type MicroFrontEndPageBindingReconciler struct {
	MicroFrontEndCommonReconciler
	RequeueDelay time.Duration
}

// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendapps,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendhosts,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagearchetypes,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagefooters,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpageheaders,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendpagenavigations,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=microfrontendrenderpages,verbs=get;list;watch;create;update;patch;delete

func (r *MicroFrontEndPageBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageBinding kdexv1alpha1.MicroFrontEndPageBinding
	if err := r.Get(ctx, req.NamespacedName, &pageBinding); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// handle finalizer
	if pageBinding.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&pageBinding, pageBindingFinalizerName) {
			controllerutil.AddFinalizer(&pageBinding, pageBindingFinalizerName)
			if err := r.Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(&pageBinding, pageBindingFinalizerName) {
			renderPage := &kdexv1alpha1.MicroFrontEndRenderPage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			}
			if err := r.Delete(ctx, renderPage); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&pageBinding, pageBindingFinalizerName)
			if err := r.Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	var host kdexv1alpha1.MicroFrontEndHost
	hostName := types.NamespacedName{
		Name:      pageBinding.Spec.HostRef.Name,
		Namespace: pageBinding.Namespace,
	}
	if err := r.Get(ctx, hostName, &host); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "referenced MicroFrontEndHost not found", "name", pageBinding.Spec.HostRef.Name)
			apimeta.SetStatusCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndHost %s not found", pageBinding.Spec.HostRef.Name),
				),
			)
			if err := r.Status().Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
		}

		log.Error(err, "unable to fetch MicroFrontEndHost", "name", pageBinding.Spec.HostRef.Name)
		return ctrl.Result{}, err
	}

	var pageArchetype kdexv1alpha1.MicroFrontEndPageArchetype
	pageArchetypeName := types.NamespacedName{
		Name:      pageBinding.Spec.PageArchetypeRef.Name,
		Namespace: pageBinding.Namespace,
	}
	if err := r.Get(ctx, pageArchetypeName, &pageArchetype); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "referenced MicroFrontEndPageArchetype not found", "name", pageBinding.Spec.PageArchetypeRef.Name)
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

			return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
		}

		log.Error(err, "unable to fetch MicroFrontEndPageArchetype", "name", pageBinding.Spec.PageArchetypeRef.Name)
		return ctrl.Result{}, err
	}

	if !apimeta.IsStatusConditionTrue(pageArchetype.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady)) {
		log.Error(fmt.Errorf("referenced MicroFrontEndPageArchetype %s is not ready", pageArchetype.Name), "")
		apimeta.SetStatusCondition(
			&pageBinding.Status.Conditions,
			*kdexv1alpha1.NewCondition(
				kdexv1alpha1.ConditionTypeReady,
				metav1.ConditionFalse,
				kdexv1alpha1.ConditionReasonReconcileError,
				fmt.Sprintf("referenced MicroFrontEndPageArchetype %s is not ready", pageArchetype.Name),
			),
		)
		if err := r.Status().Update(ctx, &pageBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
	}

	contents, response, err := r.contents(ctx, log, &pageBinding)
	if err != nil {
		return response, err
	}

	navigations, response, err := r.navigations(ctx, log, &pageBinding, &pageArchetype)
	if err != nil {
		return response, err
	}

	var parentPage kdexv1alpha1.MicroFrontEndPageBinding
	parentPageRef := pageBinding.Spec.ParentPageRef
	if parentPageRef != nil {
		parentPageName := types.NamespacedName{
			Name:      parentPageRef.Name,
			Namespace: pageBinding.Namespace,
		}

		if err := r.Get(ctx, parentPageName, &parentPage); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "referenced MicroFrontEndPageBinding not found", "name", parentPageRef.Name)
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageBinding %s not found", parentPageRef.Name),
					),
				)
				if err := r.Status().Update(ctx, &pageBinding); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageBinding", "name", parentPageRef.Name)
			return ctrl.Result{}, err
		}

		if !apimeta.IsStatusConditionTrue(parentPage.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady)) {
			log.Error(fmt.Errorf("referenced MicroFrontEndPageBinding %s is not ready", parentPage.Name), "")
			apimeta.SetStatusCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndPageBinding %s is not ready", parentPage.Name),
				),
			)
			if err := r.Status().Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
		}
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
				log.Error(err, "referenced MicroFrontEndPageHeader not found", "name", headerRef.Name)
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

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageHeader", "name", headerRef.Name)
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
				log.Error(err, "referenced MicroFrontEndPageFooter not found", "name", footerRef.Name)
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

				return ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndPageFooter", "name", footerRef.Name)
			return ctrl.Result{}, err
		}
	}

	var stylesheet kdexv1alpha1.MicroFrontEndStylesheet
	stylesheetRef := pageArchetype.Spec.OverrideStylesheetRef
	if stylesheetRef == nil {
		stylesheetRef = host.Spec.DefaultStylesheetRef
	}
	if stylesheetRef != nil {
		stylesheetName := types.NamespacedName{
			Name:      stylesheetRef.Name,
			Namespace: pageArchetype.Namespace,
		}

		if err := r.Get(ctx, stylesheetName, &stylesheet); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "referenced MicroFrontEndStylesheet not found", "name", stylesheetName.Name)
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

	renderPage := &kdexv1alpha1.MicroFrontEndRenderPage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pageBinding.Name,
			Namespace: pageBinding.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, renderPage, func() error {
		renderPage.Spec = kdexv1alpha1.MicroFrontEndRenderPageSpec{
			HostRef:         pageBinding.Spec.HostRef,
			NavigationHints: pageBinding.Spec.NavigationHints,
			PageComponents: kdexv1alpha1.PageComponents{
				Contents:        contents,
				Footer:          footer.Spec.Content,
				Header:          header.Spec.Content,
				Navigations:     navigations,
				PrimaryTemplate: pageArchetype.Spec.Content,
				Title:           pageBinding.Spec.Label,
			},
			ParentPageRef: parentPageRef,
			Paths:         pageBinding.Spec.Paths,
			StylesheetRef: stylesheetRef,
		}
		return ctrl.SetControllerReference(&pageBinding, renderPage, r.Scheme)
	}); err != nil {
		log.Error(err, "unable to create or update MicroFrontEndRenderPage")
		return ctrl.Result{}, err
	}

	log.Info("reconciled MicroFrontEndPageBinding")

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
			&kdexv1alpha1.MicroFrontEndHost{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForHost)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageArchetype{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageArchetype)).
		Watches(
			&kdexv1alpha1.MicroFrontEndPageBinding{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageBindings)).
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
				log.Error(err, "referenced MicroFrontEndApp not found", "name", appRef.Name)
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

				return nil, ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
			}

			log.Error(err, "unable to fetch MicroFrontEndApp", "name", appRef.Name)
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

			return nil, ctrl.Result{RequeueAfter: r.RequeueDelay}, nil
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
			ctx, log, *navigationRef, &pageBinding.Status.Conditions, pageBinding)

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
			ctx, log, navigationRef, &pageBinding.Status.Conditions, pageBinding)

		if navigation == nil {
			return nil, response, err
		}

		navigations[navigationName] = navigation.Spec.Content
	}

	return navigations, ctrl.Result{}, nil
}
