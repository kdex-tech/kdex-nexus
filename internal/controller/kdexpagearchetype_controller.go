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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexPageArchetypeReconciler reconciles a KDexPageArchetype object
type KDexPageArchetypeReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	RequeueDelay time.Duration
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexthemes,verbs=get;list;watch

func (r *KDexPageArchetypeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var pageArchetype kdexv1alpha1.KDexPageArchetype
	if err := r.Get(ctx, req.NamespacedName, &pageArchetype); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	kdexv1alpha1.SetConditions(
		&pageArchetype.Status.Conditions,
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
	if err := r.Status().Update(ctx, &pageArchetype); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		pageArchetype.Status.ObservedGeneration = pageArchetype.Generation
		if err := r.Status().Update(ctx, &pageArchetype); err != nil {
			log.Error(err, "failed to update pageArchetype status")
		}
	}()

	_, shouldReturn, r1, err := resolvePageFooter(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.DefaultFooterRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, r1, err = resolvePageHeader(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.DefaultHeaderRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, response, err := resolvePageNavigations(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.DefaultMainNavigationRef, pageArchetype.Spec.ExtraNavigations, r.RequeueDelay)
	if shouldReturn {
		return response, err
	}

	_, shouldReturn, r1, err = resolveScriptLibrary(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	_, shouldReturn, r1, err = resolveTheme(ctx, r.Client, &pageArchetype, &pageArchetype.Status.Conditions, pageArchetype.Spec.OverrideThemeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if err := render.ValidateContent(
		pageArchetype.Name, pageArchetype.Spec.Content,
	); err != nil {
		kdexv1alpha1.SetConditions(
			&pageArchetype.Status.Conditions,
			kdexv1alpha1.ConditionArgs{
				Degraded: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionTrue,
					Reason:  "ContentValidationFailed",
					Message: err.Error(),
				},
				Progressing: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "ContentValidationFailed",
					Message: err.Error(),
				},
				Ready: &kdexv1alpha1.ConditionFields{
					Status:  metav1.ConditionFalse,
					Reason:  "ContentValidationFailed",
					Message: err.Error(),
				},
			},
		)
		if err := r.Status().Update(ctx, &pageArchetype); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	kdexv1alpha1.SetConditions(
		&pageArchetype.Status.Conditions,
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
	if err := r.Status().Update(ctx, &pageArchetype); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexPageArchetype")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageArchetypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageArchetype{}).
		Watches(
			&kdexv1alpha1.KDexPageFooter{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageFooter)).
		Watches(
			&kdexv1alpha1.KDexPageHeader{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageHeader)).
		Watches(
			&kdexv1alpha1.KDexPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageNavigations)).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForScriptLibrary)).
		Watches(
			&kdexv1alpha1.KDexTheme{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForTheme)).
		Named("kdexpagearchetype").
		Complete(r)
}
