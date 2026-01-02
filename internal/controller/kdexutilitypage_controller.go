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
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KDexUtilityPageReconciler reconciles a KDexUtilityPage or KDexClusterUtilityPage object
type KDexUtilityPageReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexutilitypages,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexutilitypages/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexutilitypages/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterutilitypages,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterutilitypages/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterutilitypages/finalizers,verbs=update

//nolint:gocyclo
func (r *KDexUtilityPageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexUtilityPageSpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterUtilityPage kdexv1alpha1.KDexClusterUtilityPage
		if err := r.Get(ctx, req.NamespacedName, &clusterUtilityPage); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterUtilityPage.Status
		spec = clusterUtilityPage.Spec
		om = clusterUtilityPage.ObjectMeta
		o = &clusterUtilityPage
	} else {
		var utilityPage kdexv1alpha1.KDexUtilityPage
		if err := r.Get(ctx, req.NamespacedName, &utilityPage); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &utilityPage.Status
		spec = utilityPage.Spec
		om = utilityPage.ObjectMeta
		o = &utilityPage
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

		log.V(2).Info("status", "status", status, "err", err, "res", res)
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

	archetypeObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, &spec.PageArchetypeRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}
	if archetypeObj != nil {
		status.Attributes["archetype.generation"] = fmt.Sprintf("%d", archetypeObj.GetGeneration())
	}

	// Wait, ResolveContents logic is specific to KDexPageBinding.
	// It iterates `pageBinding.Spec.ContentEntries`.
	// KDexUtilityPageSpec has `ContentEntries`.
	// I should create a `ResolveUtilityPageContents` function or duplicate the small logic loop here.
	// Let's duplicate gently to avoid refactoring common code too much right now.

	for _, contentEntry := range spec.ContentEntries {
		if contentEntry.AppRef != nil {
			app, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, contentEntry.AppRef, r.RequeueDelay)
			if shouldReturn {
				return r1, err
			}
			if app != nil {
				status.Attributes[contentEntry.Slot+".content.generation"] = fmt.Sprintf("%d", app.GetGeneration())
			}
		}
	}

	headerRef := spec.OverrideHeaderRef
	headerObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, headerRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}
	if headerObj != nil {
		status.Attributes["header.generation"] = fmt.Sprintf("%d", headerObj.GetGeneration())
	}

	footerRef := spec.OverrideFooterRef
	footerObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, footerRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}
	if footerObj != nil {
		status.Attributes["footer.generation"] = fmt.Sprintf("%d", footerObj.GetGeneration())
	}

	mainNavRef := spec.OverrideMainNavigationRef
	mainNavObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, mainNavRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}
	if mainNavObj != nil {
		status.Attributes["mainNavigation.generation"] = fmt.Sprintf("%d", mainNavObj.GetGeneration())
	}

	scriptLibraryObj, shouldReturn, r1, err := ResolveKDexObjectReference(ctx, r.Client, o, &status.Conditions, spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}
	if scriptLibraryObj != nil {
		status.Attributes["scriptLibrary.generation"] = fmt.Sprintf("%d", scriptLibraryObj.GetGeneration())
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

	log.V(1).Info("reconciled")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexUtilityPageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		// Assuming webhooks will be generated eventually, similar pattern
		// err := ctrl.NewWebhookManagedBy(mgr).
		// 	For(&kdexv1alpha1.KDexUtilityPage{}).
		// 	WithDefaulter(&nexuswebhook.KDexUtilityPageDefaulter{}).
		// 	WithValidator(&nexuswebhook.KDexUtilityPageValidator{}).
		// 	Complete()

		// if err != nil {
		// 	return err
		// }
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexUtilityPage{}). // Primary watch
		Watches(
			&kdexv1alpha1.KDexClusterUtilityPage{}, // Also watch cluster scoped
			&handler.EnqueueRequestForObject{},
		).
		// Minimal Watches for dependencies - for now keep it simple to ensure basic reconciliation loop works
		// Ideally we watch all referenced objects (App, Archetype, Header, Footer...)
		WithOptions(
			controller.TypedOptions[reconcile.Request]{
				LogConstructor: LogConstructor("kdexutilitypage", mgr),
			},
		).
		Named("kdexutilitypage").
		Complete(r)
}
