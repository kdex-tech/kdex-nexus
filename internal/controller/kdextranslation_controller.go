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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const translationFinalizerName = "kdex.dev/kdex-nexus-translation-finalizer"

// KDexTranslationReconciler reconciles a KDexTranslation object
type KDexTranslationReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations/finalizers,verbs=update

func (r *KDexTranslationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var translation kdexv1alpha1.KDexTranslation
	if err := r.Get(ctx, req.NamespacedName, &translation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Defer status update
	defer func() {
		translation.Status.ObservedGeneration = translation.Generation
		if updateErr := r.Status().Update(ctx, &translation); updateErr != nil {
			err = updateErr
			res = ctrl.Result{}
		}

		log.V(1).Info("status", "status", translation.Status, "err", err, "res", res)
	}()

	if translation.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&translation, translationFinalizerName) {
			controllerutil.AddFinalizer(&translation, translationFinalizerName)
			if err := r.Update(ctx, &translation); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(&translation, translationFinalizerName) {
			// remove internal translation

			controllerutil.RemoveFinalizer(&translation, translationFinalizerName)
			if err := r.Update(ctx, &translation); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	kdexv1alpha1.SetConditions(
		&translation.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)

	_, shouldReturn, r1, err := ResolveHost(ctx, r.Client, &translation, &translation.Status.Conditions, &translation.Spec.HostRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	// create the internal translation

	kdexv1alpha1.SetConditions(
		&translation.Status.Conditions,
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
func (r *KDexTranslationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexTranslation{}).
		WithOptions(
			controller.TypedOptions[reconcile.Request]{
				LogConstructor: LogConstructor("kdextranslation", mgr),
			},
		).
		Watches(
			&kdexv1alpha1.KDexHost{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexTranslation{}, &kdexv1alpha1.KDexTranslationList{}, "{.Spec.HostRef}")).
		Named("kdextranslation").
		Complete(r)
}
