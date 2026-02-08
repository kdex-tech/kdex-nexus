package generate

import (
	"context"
	"fmt"
	"io"
	"net/http"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SHARED_VOLUME = "shared-volume"
	WORKDIR       = "/shared"
)

func CheckOrCreateGenerateJob(ctx context.Context, c client.Client, scheme *runtime.Scheme, function *kdexv1alpha1.KDexFunction, generatorConfig *kdexv1alpha1.GeneratorConfig, sa string) (*batchv1.Job, error) {
	// Create Job name
	jobName := fmt.Sprintf("%s-codegen-%d", function.Name, function.Generation)

	list := batchv1.JobList{}
	err := c.List(ctx, &list, client.InNamespace(function.Namespace), client.MatchingLabels{
		"app":        "codegen",
		"function":   function.Name,
		"generation": fmt.Sprintf("%d", function.Generation),
	})
	if err != nil {
		return nil, err
	}

	if len(list.Items) > 0 {
		return &list.Items[0], nil
	}

	url := function.Status.Attributes["openapi.schema.url.internal"]
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	functionString, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	env := []corev1.EnvVar{
		{
			Name:  "FUNCTION_NAME",
			Value: function.Name,
		},
		{
			Name:  "FUNCTION_NAMESPACE",
			Value: function.Namespace,
		},
		{
			Name:  "FUNCTION_BASEPATH",
			Value: function.Spec.API.BasePath,
		},
		{
			Name:  "FUNCTION_SPEC",
			Value: string(functionString),
		},
		{
			Name:  "COMMITTER_EMAIL",
			Value: generatorConfig.Git.CommitterEmail,
		},
		{
			Name:  "COMMITTER_NAME",
			Value: generatorConfig.Git.CommitterName,
		},
		{
			Name:  "COMMIT_SUB_DIRECTORY",
			Value: generatorConfig.Git.FunctionSubDirectory,
		},
		{
			Name: "GIT_HOST",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "host",
					LocalObjectReference: generatorConfig.Git.RepoSecretRef,
				},
			},
		},
		{
			Name: "GIT_ORG",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "org",
					LocalObjectReference: generatorConfig.Git.RepoSecretRef,
				},
			},
		},
		{
			Name: "GIT_REPO",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "repo",
					LocalObjectReference: generatorConfig.Git.RepoSecretRef,
				},
			},
		},
		{
			Name: "GIT_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "token",
					LocalObjectReference: generatorConfig.Git.RepoSecretRef,
				},
			},
		},
		{
			Name: "GIT_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "user",
					LocalObjectReference: generatorConfig.Git.RepoSecretRef,
				},
			},
		},
		{
			Name:  "WORKDIR",
			Value: WORKDIR,
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      SHARED_VOLUME,
			MountPath: WORKDIR,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: function.Namespace,
			Labels: map[string]string{
				"app":        "codegen",
				"function":   function.Name,
				"generation": fmt.Sprintf("%d", function.Generation),
			},
		},
		Spec: batchv1.JobSpec{
			Completions: utils.Ptr(int32(1)),
			Parallelism: utils.Ptr(int32(1)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: utils.Ptr(true),
					InitContainers: []corev1.Container{
						{
							Name:         "git-checkout",
							Image:        generatorConfig.Git.Image,
							Command:      []string{"git_checkout"},
							Env:          env,
							VolumeMounts: volumeMounts,
						},
						{
							Name:         "generate-code",
							Image:        generatorConfig.Image,
							Command:      generatorConfig.Command,
							Args:         generatorConfig.Args,
							Env:          env,
							VolumeMounts: volumeMounts,
						},
					},
					Containers: []corev1.Container{
						{
							Name:         "git-push",
							Image:        generatorConfig.Git.Image,
							Command:      []string{"git_push"},
							Env:          env,
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: sa,
					Volumes: []corev1.Volume{
						{
							Name: SHARED_VOLUME,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			TTLSecondsAfterFinished: utils.Ptr(int32(0)),
		},
	}

	// Add owner reference
	err = ctrl.SetControllerReference(function, job, scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to create code generation job: %w", err)
	}

	// Create the job
	err = c.Create(context.TODO(), job)
	if err != nil {
		return nil, fmt.Errorf("failed to create code generation job: %w", err)
	}

	return job, nil
}
