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
	"encoding/json"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
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
		function.Status.OpenAPISchemaURL = fmt.Sprintf("%s://%s/-/openapi?tag=%s", scheme, host.Spec.Routing.Domains[0], function.Name)
		function.Status.State = kdexv1alpha1.KDexFunctionStateOpenAPIValid
	}

	// BuildValid can happen either manually by setting spec.function.generatorConfig
	if len(function.Spec.Function.GeneratorConfig) > 0 || len(function.Status.GeneratorConfig) > 0 {
		function.Status.State = kdexv1alpha1.KDexFunctionStateBuildValid
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
	} else if function.Spec.Function.Executable == "" && function.Status.Executable == "" &&
		function.Status.URL == "" {
		// TODO: In this scenario we need to let our Build infrastructure compute the
		// StubDetails which must be set in function.Status.StubDetails and
		// function.Status.State = kdexv1alpha1.KDexFunctionStateStubGenerated

		// TODO: create a Pod that will execute the code generator using the

		// Implement code generation by creating a Job
		generatorConfig := function.Spec.Function.GeneratorConfig
		if len(generatorConfig) == 0 {
			generatorConfig = function.Status.GeneratorConfig
		}

		if len(generatorConfig) > 0 {
			// Create a Job for code generation
			job, err := createCodeGenerationJob(r.Client, &function, generatorConfig)
			if err != nil {
				kdexv1alpha1.SetConditions(
					&function.Status.Conditions,
					kdexv1alpha1.ConditionStatuses{
						Degraded:    metav1.ConditionTrue,
						Progressing: metav1.ConditionFalse,
						Ready:       metav1.ConditionFalse,
					},
					kdexv1alpha1.ConditionReasonReconcileError,
					fmt.Sprintf("Failed to create code generation job: %v", err),
				)
				return ctrl.Result{}, err
			}

			// Update status to indicate job creation
			function.Status.State = kdexv1alpha1.KDexFunctionStateBuildValid
			kdexv1alpha1.SetConditions(
				&function.Status.Conditions,
				kdexv1alpha1.ConditionStatuses{
					Degraded:    metav1.ConditionFalse,
					Progressing: metav1.ConditionTrue,
					Ready:       metav1.ConditionFalse,
				},
				kdexv1alpha1.ConditionReasonReconciling,
				fmt.Sprintf("Created code generation job %s", job.Name),
			)

			log.V(1).Info("created code generation job", "job", job.Name)
			return ctrl.Result{}, nil
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

		return ctrl.Result{}, nil
	}

	if function.Spec.Function.Executable != "" || function.Status.Executable != "" {
		function.Status.State = kdexv1alpha1.KDexFunctionStateExecutableAvailable
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

// createCodeGenerationJob creates a Kubernetes Job for code generation
func createCodeGenerationJob(client client.Client, function *kdexv1alpha1.KDexFunction, generatorConfig map[string]string) (*batchv1.Job, error) {
	// Determine the code generator to use based on generator config
	generatorImage := generatorConfig["image"]
	var args []string
	for k, v := range generatorConfig {
		if k != "image" {
			args = append(args, k)
			args = append(args, v)
		}
	}

	// Create Job name
	jobName := fmt.Sprintf("%s-codegen-%d", function.Name, time.Now().Unix())

	// Create the Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: function.Namespace,
			Labels: map[string]string{
				"app":      "codegen",
				"function": function.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: function.APIVersion,
					Kind:       "KDexFunction",
					Name:       function.Name,
					UID:        function.UID,
				},
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "generator",
							Image:   generatorImage,
							Command: []string{"/bin/sh", "-c"},
							Args:    append([]string{"echo \"Generating code for function: \"", function.Name}, args...),
							Env: []corev1.EnvVar{
								{
									Name:  "FUNCTION_NAME",
									Value: function.Name,
								},
								{
									Name:  "NAMESPACE",
									Value: function.Namespace,
								},
								{
									Name:  "FUNCTION_SPEC",
									Value: string(marshalFunctionSpec(function)),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the job
	err := client.Create(context.TODO(), job)
	if err != nil {
		return nil, fmt.Errorf("failed to create code generation job: %w", err)
	}

	return job, nil
}

func marshalFunctionSpec(fn *kdexv1alpha1.KDexFunction) []byte {
	bytes, _ := json.MarshalIndent(fn, "", "  ")
	return bytes
}
