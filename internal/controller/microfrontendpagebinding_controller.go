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

// MicroFrontEndPageBindingReconciler reconciles a MicroFrontEndPageBinding object
type MicroFrontEndPageBindingReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
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

	host, shouldReturn, r1, err := resolveHost(ctx, r.Client, &pageBinding, r.RequeueDelay)
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

	navigations, response, err := resolvePageNavigations(ctx, r.Client, &pageBinding, pageArchetype, r.RequeueDelay)
	if err != nil {
		return response, err
	}

	parentPageRef, shouldReturn, r1, err := resolveParentPageBinding(ctx, r.Client, &pageBinding, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	header, shouldReturn, r1, err := resolvePageHeader(ctx, r.Client, &pageBinding, pageArchetype, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	footer, shouldReturn, r1, err := resolvePageFooter(ctx, r.Client, &pageBinding, pageArchetype, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	stylesheetRef, shouldReturn, r1, err := resolveStylesheet(ctx, r.Client, &pageBinding, pageArchetype, host, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	renderPage := &kdexv1alpha1.MicroFrontEndRenderPage{
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
		},
	); err != nil {
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
