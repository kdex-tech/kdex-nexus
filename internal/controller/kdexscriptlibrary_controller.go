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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/npm"
	"kdex.dev/nexus/internal/validation"
	nexuswebhook "kdex.dev/nexus/internal/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KDexScriptLibraryReconciler reconciles a KDexScriptLibrary object
type KDexScriptLibraryReconciler struct {
	client.Client
	RegistryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) (npm.Registry, error)
	RequeueDelay    time.Duration
	Scheme          *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries,                  verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries/status,           verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexscriptlibraries/finalizers,       verbs=update
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterscriptlibraries,           verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterscriptlibraries/status,    verbs=get;update;patch
// +kubebuilder:rbac:groups=kdex.dev,resources=kdexclusterscriptlibraries/finalizers,verbs=update

func (r *KDexScriptLibraryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var status *kdexv1alpha1.KDexObjectStatus
	var spec kdexv1alpha1.KDexScriptLibrarySpec
	var om metav1.ObjectMeta
	var o client.Object

	if req.Namespace == "" {
		var clusterScriptLibrary kdexv1alpha1.KDexClusterScriptLibrary
		if err := r.Get(ctx, req.NamespacedName, &clusterScriptLibrary); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &clusterScriptLibrary.Status
		spec = clusterScriptLibrary.Spec
		om = clusterScriptLibrary.ObjectMeta
		o = &clusterScriptLibrary
	} else {
		var scriptLibrary kdexv1alpha1.KDexScriptLibrary
		if err := r.Get(ctx, req.NamespacedName, &scriptLibrary); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		status = &scriptLibrary.Status
		spec = scriptLibrary.Spec
		om = scriptLibrary.ObjectMeta
		o = &scriptLibrary
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

	if spec.PackageReference != nil {
		secret, shouldReturn, r1, err := ResolveSecret(ctx, r.Client, o, &status.Conditions, spec.PackageReference.SecretRef, r.RequeueDelay)
		if shouldReturn {
			return r1, err
		}

		if secret != nil {
			status.Attributes["secret.generation"] = fmt.Sprintf("%d", secret.Generation)
		}

		if err := validation.ValidatePackageReference(ctx, spec.PackageReference, secret, r.RegistryFactory); err != nil {
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
func (r *KDexScriptLibraryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if os.Getenv("ENABLE_WEBHOOKS") != FALSE {
		err := ctrl.NewWebhookManagedBy(mgr).
			For(&kdexv1alpha1.KDexScriptLibrary{}).
			WithDefaulter(&nexuswebhook.KDexScriptLibraryDefaulter{}).
			WithValidator(&nexuswebhook.KDexScriptLibraryValidator{}).
			Complete()

		if err != nil {
			return err
		}

		err = ctrl.NewWebhookManagedBy(mgr).
			For(&kdexv1alpha1.KDexClusterScriptLibrary{}).
			WithDefaulter(&nexuswebhook.KDexScriptLibraryDefaulter{}).
			WithValidator(&nexuswebhook.KDexScriptLibraryValidator{}).
			Complete()

		if err != nil {
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexScriptLibrary{}).
		Watches(
			&kdexv1alpha1.KDexClusterScriptLibrary{},
			&handler.EnqueueRequestForObject{}).
		Watches(
			&corev1.Secret{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexScriptLibrary{}, &kdexv1alpha1.KDexScriptLibraryList{}, "{.Spec.PackageReference.SecretRef}")).
		Watches(
			&corev1.Secret{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexClusterScriptLibrary{}, &kdexv1alpha1.KDexClusterScriptLibraryList{}, "{.Spec.PackageReference.SecretRef}")).
		WithOptions(controller.TypedOptions[reconcile.Request]{
			LogConstructor: LogConstructor("kdexscriptlibrary", mgr),
		}).
		Named("kdexscriptlibrary").
		Complete(r)
}
