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
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KDexTranslationReconciler reconciles a KDexTranslation object
type KDexTranslationReconciler struct {
	client.Client
	RequeueDelay time.Duration
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexinternaltranslations,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdextranslations/finalizers,verbs=update

func (r *KDexTranslationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var translation kdexv1alpha1.KDexTranslation
	if err := r.Get(ctx, req.NamespacedName, &translation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if translation.Status.Attributes == nil {
		translation.Status.Attributes = make(map[string]string)
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

	internalTranslation, err := r.createOrUpdateInternalTranslation(ctx, &translation)
	if err != nil {
		return reconcile.Result{}, err
	}

	translation.Status.Attributes = internalTranslation.Status.Attributes

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
		Owns(&kdexv1alpha1.KDexInternalTranslation{}).
		Watches(
			&kdexv1alpha1.KDexHost{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexTranslation{}, &kdexv1alpha1.KDexTranslationList{}, "{.Spec.HostRef}")).
		WithOptions(
			controller.TypedOptions[reconcile.Request]{
				LogConstructor: LogConstructor("kdextranslation", mgr)}).
		Named("kdextranslation").
		Complete(r)
}

func (r *KDexTranslationReconciler) createOrUpdateInternalTranslation(
	ctx context.Context,
	translation *kdexv1alpha1.KDexTranslation,
) (*kdexv1alpha1.KDexInternalTranslation, error) {
	log := logf.FromContext(ctx)

	internalTranslation := &kdexv1alpha1.KDexInternalTranslation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translation.Name,
			Namespace: translation.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, internalTranslation, func() error {
		if internalTranslation.CreationTimestamp.IsZero() {
			internalTranslation.Annotations = make(map[string]string)
			for key, value := range translation.Annotations {
				internalTranslation.Annotations[key] = value
			}
			internalTranslation.Labels = make(map[string]string)
			for key, value := range translation.Labels {
				internalTranslation.Labels[key] = value
			}

			internalTranslation.Labels["app.kubernetes.io/name"] = kdexWeb
			internalTranslation.Labels["kdex.dev/instance"] = translation.Name
		}

		internalTranslation.Labels["kdex.dev/generation"] = fmt.Sprintf("%d", translation.Generation)
		internalTranslation.Spec = translation.Spec

		return ctrl.SetControllerReference(translation, internalTranslation, r.Scheme)
	})

	log.V(2).Info("createOrUpdateInternalTranslation", "op", op, "err", err)

	if err != nil {
		kdexv1alpha1.SetConditions(
			&translation.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionTrue,
				Progressing: metav1.ConditionFalse,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconcileError,
			err.Error(),
		)
		return nil, err
	}

	return internalTranslation, nil
}
