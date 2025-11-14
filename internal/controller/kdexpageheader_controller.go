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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexpageheaders/finalizers,verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch

func (r *KDexPageHeaderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var pageHeader kdexv1alpha1.KDexPageHeader
	if err := r.Get(ctx, req.NamespacedName, &pageHeader); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	kdexv1alpha1.SetConditions(
		&pageHeader.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)

	// Defer status update
	defer func() {
		pageHeader.Status.ObservedGeneration = pageHeader.Generation
		if updateErr := r.Status().Update(ctx, &pageHeader); updateErr != nil {
			log.Info("failed to update status", "err", updateErr)
			if err == nil {
				err = updateErr
			}
		}
	}()

	_, shouldReturn, r1, err := ResolveScriptLibrary(ctx, r.Client, &pageHeader, &pageHeader.Status.Conditions, pageHeader.Spec.ScriptLibraryRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if err := render.ValidateContent(
		pageHeader.Name, pageHeader.Spec.Content,
	); err != nil {
		kdexv1alpha1.SetConditions(
			&pageHeader.Status.Conditions,
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
		&pageHeader.Status.Conditions,
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
			&kdexv1alpha1.KDexScriptLibrary{},
			handler.EnqueueRequestsFromMapFunc(r.findPageHeadersForScriptLibrary),
		).
		Named("kdexpageheader").
		Complete(r)
}
