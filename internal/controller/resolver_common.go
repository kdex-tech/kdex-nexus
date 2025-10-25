package controller

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/customelement"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func resolveContents(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	requeueDelay time.Duration,
) (map[string]string, ctrl.Result, error) {
	contents := make(map[string]string)

	for _, contentEntry := range pageBinding.Spec.ContentEntries {
		appRef := contentEntry.AppRef
		if appRef == nil {
			contents[contentEntry.Slot] = contentEntry.RawHTML

			continue
		}

		var app kdexv1alpha1.MicroFrontEndApp
		appName := types.NamespacedName{
			Name:      appRef.Name,
			Namespace: pageBinding.Namespace,
		}
		if err := c.Get(ctx, appName, &app); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndApp %s not found", appRef.Name),
					),
				)
				if err := c.Status().Update(ctx, pageBinding); err != nil {
					return nil, ctrl.Result{}, err
				}

				return nil, ctrl.Result{RequeueAfter: requeueDelay}, nil
			}

			return nil, ctrl.Result{}, err
		}

		if isReady, r1, err := isReady(ctx, c, pageBinding, &app, &app.Status.Conditions, requeueDelay); !isReady {
			return nil, r1, err
		}

		contents[contentEntry.Slot] = customelement.ForApp(app, contentEntry)
	}

	return contents, ctrl.Result{}, nil
}

func resolveHost(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	requeueDelay time.Duration,
) (*kdexv1alpha1.MicroFrontEndHost, bool, ctrl.Result, error) {
	var host kdexv1alpha1.MicroFrontEndHost
	hostName := types.NamespacedName{
		Name:      pageBinding.Spec.HostRef.Name,
		Namespace: pageBinding.Namespace,
	}
	if err := c.Get(ctx, hostName, &host); err != nil {
		if errors.IsNotFound(err) {
			apimeta.SetStatusCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndHost %s not found", pageBinding.Spec.HostRef.Name),
				),
			)
			if err := c.Status().Update(ctx, pageBinding); err != nil {
				return nil, true, ctrl.Result{}, err
			}

			return nil, true, ctrl.Result{RequeueAfter: requeueDelay}, nil
		}

		return nil, true, ctrl.Result{}, err
	}

	if isReady, r1, err := isReady(ctx, c, pageBinding, &host, &host.Status.Conditions, requeueDelay); !isReady {
		return nil, true, r1, err
	}

	return &host, false, ctrl.Result{}, nil
}

func resolvePageArchetype(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	requeueDelay time.Duration,
) (*kdexv1alpha1.MicroFrontEndPageArchetype, bool, ctrl.Result, error) {
	var pageArchetype kdexv1alpha1.MicroFrontEndPageArchetype
	pageArchetypeName := types.NamespacedName{
		Name:      pageBinding.Spec.PageArchetypeRef.Name,
		Namespace: pageBinding.Namespace,
	}
	if err := c.Get(ctx, pageArchetypeName, &pageArchetype); err != nil {
		if errors.IsNotFound(err) {
			apimeta.SetStatusCondition(
				&pageBinding.Status.Conditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndPageArchetype %s not found", pageBinding.Spec.PageArchetypeRef.Name),
				),
			)
			if err := c.Status().Update(ctx, pageBinding); err != nil {
				return nil, true, ctrl.Result{}, err
			}

			return nil, true, ctrl.Result{RequeueAfter: requeueDelay}, nil
		}

		return nil, true, ctrl.Result{}, err
	}

	if isReady, r1, err := isReady(ctx, c, pageBinding, &pageArchetype, &pageArchetype.Status.Conditions, requeueDelay); !isReady {
		return nil, true, r1, err
	}

	return &pageArchetype, false, ctrl.Result{}, nil
}

func resolvePageFooter(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	pageArchetype *kdexv1alpha1.MicroFrontEndPageArchetype,
	requeueDelay time.Duration,
) (*kdexv1alpha1.MicroFrontEndPageFooter, bool, ctrl.Result, error) {
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

		if err := c.Get(ctx, footerName, &footer); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageFooter %s not found", footerRef.Name),
					),
				)
				if err := c.Status().Update(ctx, pageBinding); err != nil {
					return nil, true, ctrl.Result{}, err
				}

				return nil, true, ctrl.Result{RequeueAfter: requeueDelay}, nil
			}

			return nil, true, ctrl.Result{}, err
		}

		if isReady, r1, err := isReady(ctx, c, pageBinding, &footer, &footer.Status.Conditions, requeueDelay); !isReady {
			return nil, true, r1, err
		}
	}

	return &footer, false, ctrl.Result{}, nil
}

func resolvePageHeader(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	pageArchetype *kdexv1alpha1.MicroFrontEndPageArchetype,
	requeueDelay time.Duration,
) (*kdexv1alpha1.MicroFrontEndPageHeader, bool, ctrl.Result, error) {
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

		if err := c.Get(ctx, headerName, &header); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageHeader %s not found", headerRef.Name),
					),
				)
				if err := c.Status().Update(ctx, pageBinding); err != nil {
					return nil, true, ctrl.Result{}, err
				}

				return nil, true, ctrl.Result{RequeueAfter: requeueDelay}, nil
			}

			return nil, true, ctrl.Result{}, err
		}

		if isReady, r1, err := isReady(ctx, c, pageBinding, &header, &header.Status.Conditions, requeueDelay); !isReady {
			return nil, true, r1, err
		}
	}

	return &header, false, ctrl.Result{}, nil
}

func resolvePageNavigation(
	ctx context.Context,
	c client.Client,
	object client.Object,
	objectConditions *[]metav1.Condition,
	navigationRef *v1.LocalObjectReference,
	requeueDelay time.Duration,
) (*kdexv1alpha1.MicroFrontEndPageNavigation, ctrl.Result, error) {
	var navigation kdexv1alpha1.MicroFrontEndPageNavigation
	navigationName := types.NamespacedName{
		Name:      navigationRef.Name,
		Namespace: object.GetNamespace(),
	}
	if err := c.Get(ctx, navigationName, &navigation); err != nil {
		if errors.IsNotFound(err) {
			apimeta.SetStatusCondition(
				objectConditions,
				*kdexv1alpha1.NewCondition(
					kdexv1alpha1.ConditionTypeReady,
					metav1.ConditionFalse,
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("referenced MicroFrontEndPageNavigation %s not found", navigationRef.Name),
				),
			)
			if err := c.Status().Update(ctx, object); err != nil {
				return nil, ctrl.Result{}, err
			}

			return nil, ctrl.Result{RequeueAfter: requeueDelay}, nil
		}

		return nil, ctrl.Result{}, err
	}

	if isReady, r1, err := isReady(ctx, c, object, &navigation, &navigation.Status.Conditions, requeueDelay); !isReady {
		return nil, r1, err
	}

	return &navigation, ctrl.Result{}, nil
}

func resolvePageNavigations(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	pageArchetype *kdexv1alpha1.MicroFrontEndPageArchetype,
	requeueDelay time.Duration,
) (map[string]string, ctrl.Result, error) {
	navigations := make(map[string]string)

	navigationRef := pageBinding.Spec.OverrideMainNavigationRef
	if navigationRef == nil {
		navigationRef = pageArchetype.Spec.DefaultMainNavigationRef
	}
	if navigationRef != nil {
		navigation, response, err := resolvePageNavigation(
			ctx, c, pageBinding, &pageBinding.Status.Conditions, navigationRef, requeueDelay)

		if navigation == nil {
			return nil, response, err
		}

		navigations["main"] = navigation.Spec.Content
	}

	if pageArchetype.Spec.ExtraNavigations == nil {
		return navigations, ctrl.Result{}, nil
	}

	for navigationName, navigationRef := range *pageArchetype.Spec.ExtraNavigations {
		navigation, response, err := resolvePageNavigation(
			ctx, c, pageBinding, &pageBinding.Status.Conditions, &navigationRef, requeueDelay)

		if navigation == nil {
			return nil, response, err
		}

		navigations[navigationName] = navigation.Spec.Content
	}

	return navigations, ctrl.Result{}, nil
}

func resolveParentPageBinding(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	requeueDelay time.Duration,
) (*v1.LocalObjectReference, bool, ctrl.Result, error) {
	var parentPage kdexv1alpha1.MicroFrontEndPageBinding
	parentPageRef := pageBinding.Spec.ParentPageRef
	if parentPageRef != nil {
		parentPageName := types.NamespacedName{
			Name:      parentPageRef.Name,
			Namespace: pageBinding.Namespace,
		}

		if err := c.Get(ctx, parentPageName, &parentPage); err != nil {
			if errors.IsNotFound(err) {
				apimeta.SetStatusCondition(
					&pageBinding.Status.Conditions,
					*kdexv1alpha1.NewCondition(
						kdexv1alpha1.ConditionTypeReady,
						metav1.ConditionFalse,
						kdexv1alpha1.ConditionReasonReconcileError,
						fmt.Sprintf("referenced MicroFrontEndPageBinding %s not found", parentPageRef.Name),
					),
				)
				if err := c.Status().Update(ctx, pageBinding); err != nil {
					return nil, true, ctrl.Result{}, err
				}

				return nil, true, ctrl.Result{RequeueAfter: requeueDelay}, nil
			}

			return nil, true, ctrl.Result{}, err
		}

		if isReady, r1, err := isReady(ctx, c, pageBinding, &parentPage, &parentPage.Status.Conditions, requeueDelay); !isReady {
			return nil, true, r1, err
		}
	}

	return parentPageRef, false, ctrl.Result{}, nil
}

func resolveStylesheet(
	ctx context.Context,
	c client.Client,
	pageBinding *kdexv1alpha1.MicroFrontEndPageBinding,
	pageArchetype *kdexv1alpha1.MicroFrontEndPageArchetype,
	host *kdexv1alpha1.MicroFrontEndHost,
	requeueDelay time.Duration,
) (*v1.LocalObjectReference, bool, ctrl.Result, error) {
	stylesheetRef := pageArchetype.Spec.OverrideStylesheetRef
	if stylesheetRef == nil {
		stylesheetRef = host.Spec.DefaultStylesheetRef
	}
	if stylesheetRef != nil {
		var stylesheet kdexv1alpha1.MicroFrontEndStylesheet
		stylesheetName := types.NamespacedName{
			Name:      stylesheetRef.Name,
			Namespace: pageBinding.Namespace,
		}
		if err := c.Get(ctx, stylesheetName, &stylesheet); err != nil {
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
				if err := c.Status().Update(ctx, pageBinding); err != nil {
					return nil, true, ctrl.Result{}, err
				}

				return nil, true, ctrl.Result{RequeueAfter: requeueDelay}, nil
			}

			return nil, true, ctrl.Result{}, err
		}

		if isReady, r1, err := isReady(ctx, c, pageBinding, &stylesheet, &stylesheet.Status.Conditions, requeueDelay); !isReady {
			return nil, true, r1, err
		}
	}

	return stylesheetRef, false, ctrl.Result{}, nil
}
