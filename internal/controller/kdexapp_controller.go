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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/npm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KDexAppReconciler reconciles a KDexApp object
type KDexAppReconciler struct {
	client.Client
	RegistryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) (npm.Registry, error)
	RequeueDelay    time.Duration
	Scheme          *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,                       verbs=get;list;watch

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps,                  verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps/status,           verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexapps/finalizers,       verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterapps,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterapps/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterapps/finalizers,verbs=update

func (r *KDexAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexAppSpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterApp kdexv1alpha1.KDexClusterApp
		if err := r.Get(ctx, req.NamespacedName, &clusterApp); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterApp.Status
		spec = clusterApp.Spec
		om = clusterApp.ObjectMeta
		o = &clusterApp
	} else {
		var app kdexv1alpha1.KDexApp
		if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &app.Status
		spec = app.Spec
		om = app.ObjectMeta
		o = &app
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

	secret, shouldReturn, r1, err := ResolveSecret(ctx, r.Client, o, &status.Conditions, spec.PackageReference.SecretRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	if secret != nil {
		status.Attributes["secret.generation"] = fmt.Sprintf("%d", secret.Generation)
	}

	if err := validatePackageReference(ctx, &spec.PackageReference, secret, r.RegistryFactory); err != nil {
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

	log.Info("reconciled KDexApp")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KDexAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexApp{}).
		Watches(
			&kdexv1alpha1.KDexClusterApp{},
			LikeNamedHandler).
		Watches(
			&corev1.Secret{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexApp{}, &kdexv1alpha1.KDexAppList{}, "{.Spec.PackageReference.SecretRef}")).
		Named("kdexapp").
		Complete(r)
}
