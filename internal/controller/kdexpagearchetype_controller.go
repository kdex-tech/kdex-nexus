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
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KDexPageArchetypeReconciler reconciles a KDexPageArchetype object
type KDexPageArchetypeReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	RequeueDelay time.Duration
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes,                  verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes/status,           verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagearchetypes/finalizers,       verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagearchetypes,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagearchetypes/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpagearchetypes/finalizers,verbs=update

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagefooters,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpagenavigations,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

func (r *KDexPageArchetypeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexPageArchetypeSpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterPageArchetype kdexv1alpha1.KDexClusterPageArchetype
		if err := r.Get(ctx, req.NamespacedName, &clusterPageArchetype); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterPageArchetype.Status
		spec = clusterPageArchetype.Spec
		om = clusterPageArchetype.ObjectMeta
		o = &clusterPageArchetype
	} else {
		var pageArchetype kdexv1alpha1.KDexPageArchetype
		if err := r.Get(ctx, req.NamespacedName, &pageArchetype); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &pageArchetype.Status
		spec = pageArchetype.Spec
		om = pageArchetype.ObjectMeta
		o = &pageArchetype
	}

	if status.Attributes == nil {
		status.Attributes = make(map[string]string)
	}

	// Defer status update
	defer func() {
		status.ObservedGeneration = om.Generation
		if updateErr := r.Status().Update(ctx, o); updateErr != nil {
			err = updateErr
			res = ctrl.Result{}
		}
	}()

	kdexv1alpha1.SetConditions(
		&status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)

	footerObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, spec.DefaultFooterRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if footerObj != nil {
		status.Attributes["footer.generation"] = fmt.Sprintf("%d", footerObj.GetGeneration())
	}

	headerObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, spec.DefaultHeaderRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if headerObj != nil {
		status.Attributes["header.generation"] = fmt.Sprintf("%d", headerObj.GetGeneration())
	}

	navigations, shouldReturn, response, err := ResolvePageNavigations(ctx, r.Client, o, &status.Conditions, spec.DefaultMainNavigationRef, spec.ExtraNavigations, r.RequeueDelay)
	if shouldReturn {
		return response, err
	}

	for k, navigation := range navigations {
		status.Attributes[k+".navigation.generation"] = fmt.Sprintf("%d", navigation.GetGeneration())
	}

	scriptLibraryObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if scriptLibraryObj != nil {
		status.Attributes["scriptLibrary.generation"] = fmt.Sprintf("%d", scriptLibraryObj.GetGeneration())
	}

	if err := render.ValidateContent(
		o.GetName(), spec.Content,
	); err != nil {
		kdexv1alpha1.SetConditions(
			&status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)

		return ctrl.Result{}, err
	}

	kdexv1alpha1.SetConditions(
		&status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionFalse,
			Ready:       metav1.ConditionTrue,
		},
		kdexv1alpha1.ConditionReasonReconcileSuccess,
		"Reconciliation successful",
	)

	log.Info("reconciled KDexPageArchetype")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageArchetypeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageArchetype{}).
		Watches(
			&kdexv1alpha1.KDexClusterPageArchetype{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: o.GetName()}}}
			}),
		).
		Watches(
			&kdexv1alpha1.KDexPageFooter{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageFooter)).
		Watches(
			&kdexv1alpha1.KDexClusterPageFooter{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageFooter)).
		Watches(
			&kdexv1alpha1.KDexPageHeader{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageHeader)).
		Watches(
			&kdexv1alpha1.KDexClusterPageHeader{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageHeader)).
		Watches(
			&kdexv1alpha1.KDexPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageNavigations)).
		Watches(
			&kdexv1alpha1.KDexClusterPageNavigation{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForPageNavigations)).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForScriptLibrary)).
		Watches(
			&kdexv1alpha1.KDexClusterScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageArchetypesForScriptLibrary)).
		Named("kdexpagearchetype").
		Complete(r)
}
