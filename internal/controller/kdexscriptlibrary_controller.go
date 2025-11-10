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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/npm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexScriptLibraryReconciler reconciles a KDexScriptLibrary object
type KDexScriptLibraryReconciler struct {
	client.Client
	RegistryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) npm.Registry
	RequeueDelay    time.Duration
	Scheme          *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries/finalizers,verbs=update

func (r *KDexScriptLibraryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var scriptLibrary kdexv1alpha1.KDexScriptLibrary
	if err := r.Get(ctx, req.NamespacedName, &scriptLibrary); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	kdexv1alpha1.SetConditions(
		&scriptLibrary.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		"Reconciling",
	)
	if err := r.Status().Update(ctx, &scriptLibrary); err != nil {
		return ctrl.Result{}, err
	}

	// Defer status update
	defer func() {
		scriptLibrary.Status.ObservedGeneration = scriptLibrary.Generation
		if err := r.Status().Update(ctx, &scriptLibrary); err != nil {
			log.Error(err, "failed to update app status")
		}
	}()

	if scriptLibrary.Spec.PackageReference != nil {
		secret, shouldReturn, r1, err := resolveSecret(ctx, r.Client, &scriptLibrary, &scriptLibrary.Status.Conditions, scriptLibrary.Spec.PackageReference.SecretRef, r.RequeueDelay)
		if shouldReturn {
			return r1, err
		}

		if err := validatePackageReference(ctx, scriptLibrary.Spec.PackageReference, secret, r.RegistryFactory); err != nil {
			kdexv1alpha1.SetConditions(
				&scriptLibrary.Status.Conditions,
				kdexv1alpha1.ConditionStatuses{
					Degraded:    metav1.ConditionTrue,
					Progressing: metav1.ConditionFalse,
					Ready:       metav1.ConditionFalse,
				},
				kdexv1alpha1.ConditionReasonReconcileError,
				err.Error(),
			)
			if err := r.Status().Update(ctx, &scriptLibrary); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}

	if scriptLibrary.Spec.Scripts != nil {
		if err := validateScripts(&scriptLibrary.Spec); err != nil {
			kdexv1alpha1.SetConditions(
				&scriptLibrary.Status.Conditions,
				kdexv1alpha1.ConditionStatuses{
					Degraded:    metav1.ConditionTrue,
					Progressing: metav1.ConditionFalse,
					Ready:       metav1.ConditionFalse,
				},
				kdexv1alpha1.ConditionReasonReconcileError,
				err.Error(),
			)
			if err := r.Status().Update(ctx, &scriptLibrary); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}
	}

	kdexv1alpha1.SetConditions(
		&scriptLibrary.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionFalse,
			Ready:       metav1.ConditionTrue,
		},
		kdexv1alpha1.ConditionReasonReconcileSuccess,
		"Reconciliation successful",
	)
	if err := r.Status().Update(ctx, &scriptLibrary); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled KDexScriptLibrary")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexScriptLibraryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexScriptLibrary{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findScriptLibrariesForSecret),
		).
		Named("kdexscriptlibrary").
		Complete(r)
}
