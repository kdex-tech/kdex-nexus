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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const pageBindingFinalizerName = "kdex.dev/kdex-nexus-page-binding-finalizer"

// KDexPageBindingReconciler reconciles a KDexPageBinding object
type KDexPageBindingReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	RequeueDelay time.Duration
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexhosts,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexrenderpages,verbs=get;list;watch;create;update;patch;delete

func (r *KDexPageBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageBinding kdexv1alpha1.KDexPageBinding
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
			renderPage := &kdexv1alpha1.KDexRenderPage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			}
			if err := r.Delete(ctx, renderPage); client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&pageBinding, pageBindingFinalizerName)
			if err := r.Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	_, shouldReturn, r1, err := resolveHost(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, &pageBinding.Spec.HostRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	pageArchetype, shouldReturn, r1, err := resolvePageArchetype(ctx, r.Client, &pageBinding, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	contents, response, err := resolveContents(ctx, r.Client, &pageBinding, r.RequeueDelay)
	if err != nil {
		return response, err
	}

	navigationRef := pageBinding.Spec.OverrideMainNavigationRef
	if navigationRef == nil {
		navigationRef = pageArchetype.Spec.DefaultMainNavigationRef
	}
	navigations, response, err := resolvePageNavigations(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, navigationRef, pageArchetype.Spec.ExtraNavigations, r.RequeueDelay)
	if err != nil {
		return response, err
	}

	parentPageRef, shouldReturn, r1, err := resolvePageBinding(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, pageBinding.Spec.ParentPageRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	headerRef := pageBinding.Spec.OverrideHeaderRef
	if headerRef == nil {
		headerRef = pageArchetype.Spec.DefaultHeaderRef
	}
	header, shouldReturn, r1, err := resolvePageHeader(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, headerRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	footerRef := pageBinding.Spec.OverrideFooterRef
	if footerRef == nil {
		footerRef = pageArchetype.Spec.DefaultFooterRef
	}
	footer, shouldReturn, r1, err := resolvePageFooter(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, footerRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, r1, err = resolveTheme(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, pageArchetype.Spec.OverrideThemeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	renderPage := &kdexv1alpha1.KDexRenderPage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pageBinding.Name,
			Namespace: pageBinding.Namespace,
		},
	}

	if _, err := ctrl.CreateOrUpdate(
		ctx,
		r.Client,
		renderPage,
		func() error {
			renderPage.Spec = kdexv1alpha1.KDexRenderPageSpec{
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
				ThemeRef:      pageArchetype.Spec.OverrideThemeRef,
			}
			return ctrl.SetControllerReference(&pageBinding, renderPage, r.Scheme)
		},
	); err != nil {
		log.Error(err, "unable to create or update KDexRenderPage")
		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexPageBinding")

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
func (r *KDexPageBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageBinding{}).
		Owns(&kdexv1alpha1.KDexRenderPage{}).
		Watches(
			&kdexv1alpha1.KDexApp{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForApp)).
		Watches(
			&kdexv1alpha1.KDexHost{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForHost)).
		Watches(
			&kdexv1alpha1.KDexPageArchetype{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageArchetype)).
		Watches(
			&kdexv1alpha1.KDexPageBinding{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageBindings)).
		Watches(
			&kdexv1alpha1.KDexPageFooter{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageFooter)).
		Watches(
			&kdexv1alpha1.KDexPageHeader{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageHeader)).
		Watches(
			&kdexv1alpha1.KDexPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForPageNavigations)).
		Named("kdexpagebinding").
		Complete(r)
}
