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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	"kdex.dev/nexus/internal/generate"
	"kdex.dev/nexus/internal/utils"
	nexuswebhook "kdex.dev/nexus/internal/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KDexFunctionReconciler reconciles a KDexFunction object
type KDexFunctionReconciler struct {
	client.Client
	Configuration configuration.NexusConfiguration
	RequeueDelay  time.Duration
	Scheme        *runtime.Scheme
}

func (r *KDexFunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := logf.FromContext(ctx)

	var function kdexv1alpha1.KDexFunction
	if err := r.Get(ctx, req.NamespacedName, &function); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if function.Status.Attributes == nil {
		function.Status.Attributes = make(map[string]string)
	}

	// Defer status update
	defer func() {
		function.Status.ObservedGeneration = function.Generation
		if updateErr := r.Status().Update(ctx, &function); updateErr != nil {
			err = updateErr
			res = ctrl.Result{}
		}

		log.V(2).Info("status", "status", function.Status, "err", err, "res", res)
	}()

	kdexv1alpha1.SetConditions(
		&function.Status.Conditions,
		kdexv1alpha1.ConditionStatuses{
			Degraded:    metav1.ConditionFalse,
			Progressing: metav1.ConditionTrue,
			Ready:       metav1.ConditionUnknown,
		},
		kdexv1alpha1.ConditionReasonReconciling,
		string(kdexv1alpha1.KDexFunctionStatePending),
	)

	host, shouldReturn, r1, err := ResolveHost(ctx, r.Client, &function, &function.Status.Conditions, &function.Spec.HostRef, r.RequeueDelay)
	if shouldReturn {
		return r1, err
	}

	// OpenAPIValid should result purely through validation webhook
	if function.Status.OpenAPISchemaURL == "" {
		scheme := "http"
		if host.Spec.Routing.TLS != nil {
			scheme = "https"
		}
		function.Status.OpenAPISchemaURL = fmt.Sprintf("%s://%s/-/openapi?type=function&tag=%s", scheme, host.Spec.Routing.Domains[0], function.Name)
		function.Status.State = kdexv1alpha1.KDexFunctionStateOpenAPIValid
		function.Status.Detail = "OpenAPISchemaURL:" + function.Status.OpenAPISchemaURL
	}

	// BuildValid can happen either manually by setting spec.function.generatorConfig
	if function.Spec.Function.GeneratorConfig != nil || function.Status.GeneratorConfig != nil {
		function.Status.State = kdexv1alpha1.KDexFunctionStateBuildValid
		if function.Spec.Function.GeneratorConfig != nil {
			function.Status.Detail = "GeneratorImage:" + function.Spec.Function.GeneratorConfig.Image
		}
		if function.Status.GeneratorConfig != nil {
			function.Status.Detail = "GeneratorImage:" + function.Status.GeneratorConfig.Image
		}
	} else if function.Spec.Function.StubDetails == nil && function.Status.StubDetails == nil &&
		function.Spec.Function.Executable == "" && function.Status.Executable == "" &&
		function.Status.URL == "" {

		// TODO: In this scenario we need to let our Build infrastructure compute the
		// GeneratorConfig which must be set in function.Status.GeneratorConfig
		// and function.Status.State = kdexv1alpha1.KDexFunctionStateBuildValid
		kdexv1alpha1.SetConditions(
			&function.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionFalse,
				Progressing: metav1.ConditionTrue,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconciling,
			string(kdexv1alpha1.KDexFunctionStateOpenAPIValid),
		)

		log.V(1).Info(string(kdexv1alpha1.KDexFunctionStateOpenAPIValid))

		return ctrl.Result{}, nil
	}

	if function.Spec.Function.StubDetails != nil || function.Status.StubDetails != nil {
		function.Status.State = kdexv1alpha1.KDexFunctionStateStubGenerated
		if function.Spec.Function.StubDetails != nil {
			function.Status.Detail = "StubSource:" + utils.IfElse(
				function.Spec.Function.StubDetails.SourceImage != "",
				function.Spec.Function.StubDetails.SourceImage,
				function.Spec.Function.StubDetails.SourcePath,
			)
		}
		if function.Status.StubDetails != nil {
			function.Status.Detail = "StubSource:" + utils.IfElse(
				function.Status.StubDetails.SourceImage != "",
				function.Status.StubDetails.SourceImage,
				function.Status.StubDetails.SourcePath,
			)
		}
	} else if function.Spec.Function.Executable == "" && function.Status.Executable == "" &&
		function.Status.URL == "" {
		// TODO: In this scenario we need to let our Build infrastructure compute the
		// StubDetails which must be set in function.Status.StubDetails and
		// function.Status.State = kdexv1alpha1.KDexFunctionStateStubGenerated

		generatorConfig := function.Spec.Function.GeneratorConfig
		if generatorConfig == nil {
			generatorConfig = function.Status.GeneratorConfig
		}

		if generatorConfig.Image == "" {
			err := fmt.Errorf("GeneratorConfig image empty: %s/%s", function.Namespace, function.Name)
			kdexv1alpha1.SetConditions(
				&function.Status.Conditions,
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

		job, err := generate.CheckOrCreateGenerateJob(ctx, r.Client, r.Scheme, &function, generatorConfig, host.Name)
		if err != nil {
			kdexv1alpha1.SetConditions(
				&function.Status.Conditions,
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

		if job != nil {
			for _, cond := range job.Status.Conditions {
				if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
					err := fmt.Errorf("Code generation job failed: %s", cond.Message)
					kdexv1alpha1.SetConditions(
						&function.Status.Conditions,
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
		}

		kdexv1alpha1.SetConditions(
			&function.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionFalse,
				Progressing: metav1.ConditionTrue,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconciling,
			string(kdexv1alpha1.KDexFunctionStateBuildValid),
		)

		log.V(1).Info(string(kdexv1alpha1.KDexFunctionStateBuildValid))

		function.Status.State = kdexv1alpha1.KDexFunctionStateBuildValid
		function.Status.Detail = "GeneratorImage:" + generatorConfig.Image

		return ctrl.Result{}, nil
	}

	if function.Spec.Function.Executable != "" || function.Status.Executable != "" {
		function.Status.State = kdexv1alpha1.KDexFunctionStateExecutableAvailable
		function.Status.Detail = "Executable:" + utils.IfElse(
			function.Spec.Function.Executable != "",
			function.Spec.Function.Executable,
			function.Status.Executable,
		)
	} else if function.Status.URL == "" {
		// TODO: In this scenario we need to let our Build infrastructure create the
		// Executable which must be set in function.Status.Executable and
		// function.Status.State = kdexv1alpha1.KDexFunctionStateExecutableAvailable
		kdexv1alpha1.SetConditions(
			&function.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionFalse,
				Progressing: metav1.ConditionTrue,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconciling,
			string(kdexv1alpha1.KDexFunctionStateStubGenerated),
		)

		log.V(1).Info(string(kdexv1alpha1.KDexFunctionStateStubGenerated))

		return ctrl.Result{}, nil
	}

	if function.Status.URL != "" {
		function.Status.State = kdexv1alpha1.KDexFunctionStateFunctionDeployed
		function.Status.Detail = "FunctionURL:" + function.Status.URL
	} else {
		// TODO: In this scenario we need to trigger the function deployment and
		// wait for it to reconcile, then set the URL on function.Status.URL and
		// function.Status.State = kdexv1alpha1.KDexFunctionStateFunctionDeployed
		kdexv1alpha1.SetConditions(
			&function.Status.Conditions,
			kdexv1alpha1.ConditionStatuses{
				Degraded:    metav1.ConditionFalse,
				Progressing: metav1.ConditionTrue,
				Ready:       metav1.ConditionFalse,
			},
			kdexv1alpha1.ConditionReasonReconciling,
			string(kdexv1alpha1.KDexFunctionStateExecutableAvailable),
		)

		log.V(1).Info(string(kdexv1alpha1.KDexFunctionStateExecutableAvailable))

		return ctrl.Result{}, nil
	}

	kdexv1alpha1.SetConditions(
		&function.Status.Conditions,
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
func (r *KDexFunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if os.Getenv("ENABLE_WEBHOOKS") != FALSE {
		err := ctrl.NewWebhookManagedBy(mgr, &kdexv1alpha1.KDexFunction{}).
			WithDefaulter(&nexuswebhook.KDexFunctionDefaulter[*kdexv1alpha1.KDexFunction]{}).
			WithValidator(&nexuswebhook.KDexFunctionValidator[*kdexv1alpha1.KDexFunction]{}).
			Complete()

		if err != nil {
			return err
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kdexv1alpha1.KDexFunction{}).
		Watches(
			&kdexv1alpha1.KDexHost{},
			MakeHandlerByReferencePath(r.Client, r.Scheme, &kdexv1alpha1.KDexFunction{}, &kdexv1alpha1.KDexFunctionList{}, "{.Spec.HostRef}")).
		WithOptions(controller.TypedOptions[reconcile.Request]{
			LogConstructor: LogConstructor("kdexfunction", mgr),
		}).
		Named("kdexfunction").
		Complete(r)
}
