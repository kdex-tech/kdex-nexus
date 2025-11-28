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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
)

// KDexPageHeaderReconciler reconciles a KDexPageHeader object
type KDexPageHeaderReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders,                  verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders/status,           verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders/finalizers,       verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpageheaders,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpageheaders/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterpageheaders/finalizers,verbs=update

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

func (r *KDexPageHeaderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexPageHeaderSpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterPageHeader kdexv1alpha1.KDexClusterPageHeader
		if err := r.Get(ctx, req.NamespacedName, &clusterPageHeader); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterPageHeader.Status
		spec = clusterPageHeader.Spec
		o = &clusterPageHeader
	} else {
		var pageHeader kdexv1alpha1.KDexPageHeader
		if err := r.Get(ctx, req.NamespacedName, &pageHeader); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &pageHeader.Status
		spec = pageHeader.Spec
		o = &pageHeader
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

	log.Info("reconciled KDexPageHeader")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexPageHeaderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexPageHeader{}).
		Watches(
			&kdexv1alpha1.KDexClusterPageHeader{},
			LikeNamedHandler).
		Watches(
			&kdexv1alpha1.KDexScriptLibrary{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexPageHeader{}, &kdexv1alpha1.KDexPageHeaderList{}, "{.Spec.ScriptLibraryRef}")).
		Watches(
			&kdexv1alpha1.KDexClusterScriptLibrary{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexPageHeader{}, &kdexv1alpha1.KDexPageHeaderList{}, "{.Spec.ScriptLibraryRef}")).
		Named("kdexpageheader").
		Complete(r)
}
