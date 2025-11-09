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
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
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
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

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
			// TODO

			controllerutil.RemoveFinalizer(&pageBinding, pageBindingFinalizerName)
			if err := r.Update(ctx, &pageBinding); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	kdexv1alpha1.SetConditions(
		&pageBinding.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionUnknown,
				Reason:  kdexv1alpha1.ConditionReasonReconciling,
				Message: "Reconciling",
			},
		},
	)
	if err := r.Status().Update(ctx, &pageBinding); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		pageBinding.Status.ObservedGeneration = pageBinding.Generation
		if err := r.Status().Update(ctx, &pageBinding); err != nil {
			log.Error(err, "failed to update pageBinding status")
		}
	}()

	host, shouldReturn, r1, err := resolveHost(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, &pageBinding.Spec.HostRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	archetype, shouldReturn, r1, err := resolvePageArchetype(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, &pageBinding.Spec.PageArchetypeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	contents, shouldReturn, response, err := resolveContents(ctx, r.Client, &pageBinding, r.RequeueDelay)
	if shouldReturn {
		return response, err
	}

	navigationRef := pageBinding.Spec.OverrideMainNavigationRef
	if navigationRef == nil {
		navigationRef = archetype.Spec.DefaultMainNavigationRef
	}
	navigations, shouldReturn, response, err := resolvePageNavigations(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, navigationRef, archetype.Spec.ExtraNavigations, r.RequeueDelay)
	if shouldReturn {
		return response, err
	}

	parentBinding, shouldReturn, r1, err := resolvePageBinding(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, pageBinding.Spec.ParentPageRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	headerRef := pageBinding.Spec.OverrideHeaderRef
	if headerRef == nil {
		headerRef = archetype.Spec.DefaultHeaderRef
	}
	header, shouldReturn, r1, err := resolvePageHeader(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, headerRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	footerRef := pageBinding.Spec.OverrideFooterRef
	if footerRef == nil {
		footerRef = archetype.Spec.DefaultFooterRef
	}
	footer, shouldReturn, r1, err := resolvePageFooter(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, footerRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	scriptLibrary, shouldReturn, r1, err := resolveScriptLibrary(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, pageBinding.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	theme, shouldReturn, r1, err := resolveTheme(ctx, r.Client, &pageBinding, &pageBinding.Status.Conditions, archetype.Spec.OverrideThemeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if pageBinding.Spec.BasePath == "/" && pageBinding.Spec.ParentPageRef != nil {
		err := fmt.Errorf("a page binding with basePath set to '/' must not specify a parent page binding")

		kdexv1alpha1.SetConditions(
			&pageBinding.Status.Conditions,
			kdexv1alpha1.ConditionArgs{
				Degraded: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionTrue,
					Reason:  "SpecValidationFailed",
					Message: err.Error(),
				},
				Progressing: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "SpecValidationFailed",
					Message: err.Error(),
				},
				Ready: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "SpecValidationFailed",
					Message: err.Error(),
				},
			},
		)
		if err := r.Status().Update(ctx, &pageBinding); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	resolved := ResolvedPageBinding{
		Archetype:     *archetype,
		Contents:      contents,
		Footer:        footer,
		Header:        header,
		Host:          *host,
		Navigations:   navigations,
		PageBinding:   &pageBinding,
		ParentBinding: parentBinding,
		ScriptLibrary: scriptLibrary,
		Theme:         theme,
	}

	err = r._reconcile(ctx, resolved)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageBinding{}).
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
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageBindingsForScriptLibrary)).
		Named("kdexpagebinding").
		Complete(r)
}

type ResolvedPageBinding struct {
	Archetype     kdexv1alpha1.KDexPageArchetype
	Contents      map[string]ResolvedContentEntry
	Footer        *kdexv1alpha1.KDexPageFooter
	Header        *kdexv1alpha1.KDexPageHeader
	Host          kdexv1alpha1.KDexHost
	Navigations   map[string]*kdexv1alpha1.KDexPageNavigation
	PageBinding   *kdexv1alpha1.KDexPageBinding
	ParentBinding *kdexv1alpha1.KDexPageBinding
	ScriptLibrary *kdexv1alpha1.KDexScriptLibrary
	Theme         *kdexv1alpha1.KDexTheme
}

func (r *KDexPageBindingReconciler) _reconcile(ctx context.Context, resolved ResolvedPageBinding) error {
	log := logf.FromContext(ctx)

	log.Info("host", "host", resolved.Host)

	log.Info("archetype", "archetype", resolved.Archetype)

	log.Info("contents", "contents", resolved.Contents)

	log.Info("navigations", "navigations", resolved.Navigations)

	log.Info("parentBinding", "parentBinding", resolved.ParentBinding)

	log.Info("header", "header", resolved.Header)

	log.Info("footer", "footer", resolved.Footer)

	log.Info("scriptLibrary", "scriptLibrary", resolved.ScriptLibrary)

	log.Info("theme", "theme", resolved.Theme)

	// scriptLibraryRefs := []corev1.LocalObjectReference{}

	// if pageArchetype.Spec.ScriptLibraryRef != nil {
	// 	scriptLibraryRefs = append(scriptLibraryRefs, *pageArchetype.Spec.ScriptLibraryRef)
	// }

	// if header != nil && header.Spec.ScriptLibraryRef != nil {
	// 	scriptLibraryRefs = append(scriptLibraryRefs, *header.Spec.ScriptLibraryRef)
	// }

	// if footer != nil && footer.Spec.ScriptLibraryRef != nil {
	// 	scriptLibraryRefs = append(scriptLibraryRefs, *footer.Spec.ScriptLibraryRef)
	// }

	// if pageBinding.Spec.ScriptLibraryRef != nil {
	// 	scriptLibraryRefs = append(scriptLibraryRefs, *pageBinding.Spec.ScriptLibraryRef)
	// }

	// log.Info("scriptLibraryRefs", "scriptLibraryRefs", scriptLibraryRefs)

	// renderPage := &kdexv1alpha1.KDexRenderPage{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      pageBinding.Name,
	// 		Namespace: pageBinding.Namespace,
	// 	},
	// }

	// if _, err := ctrl.CreateOrUpdate(
	// 	ctx,
	// 	r.Client,
	// 	renderPage,
	// 	func() error {

	// 		renderPage.Spec = kdexv1alpha1.KDexRenderPageSpec{
	// 			HostRef:         pageBinding.Spec.HostRef,
	// 			NavigationHints: pageBinding.Spec.NavigationHints,
	// 			PageComponents: kdexv1alpha1.PageComponents{
	// 				// Contents:        contents,
	// 				Footer: footer.Spec.Content,
	// 				Header: header.Spec.Content,
	// 				// Navigations:     navigations,
	// 				PrimaryTemplate: pageArchetype.Spec.Content,
	// 				Title:           pageBinding.Spec.Label,
	// 			},
	// 			ParentPageRef:     pageBinding.Spec.ParentPageRef,
	// 			Paths:             pageBinding.Spec.Paths,
	// 			ScriptLibraryRefs: scriptLibraryRefs,
	// 			ThemeRef:          pageArchetype.Spec.OverrideThemeRef,
	// 		}
	// 		return ctrl.SetControllerReference(&pageBinding, renderPage, r.Scheme)
	// 	},
	// ); err != nil {
	// 	log.Error(err, "unable to create or update KDexRenderPage")
	// 	return ctrl.Result{}, err
	// }

	kdexv1alpha1.SetConditions(
		&resolved.PageBinding.Status.Conditions,
		kdexv1alpha1.ConditionArgs{
			Degraded: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
			Progressing: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionFalse,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
			Ready: &kdexv1alpha1.ConditionFields{
				Status:  metav1.ConditionTrue,
				Reason:  kdexv1alpha1.ConditionReasonReconcileSuccess,
				Message: "Reconciliation successful",
			},
		},
	)
	if err := r.Status().Update(ctx, resolved.PageBinding); err != nil {
		return err
	}

	log.Info("reconciled KDexPageBinding")

	return nil
}
