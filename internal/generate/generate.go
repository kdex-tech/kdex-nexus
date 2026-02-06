package generate

import (
	"context"
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/api/resource"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckOrCreateGenerateJob(ctx context.Context, c client.Client, scheme *runtime.Scheme, function *kdexv1alpha1.KDexFunction, sa string) (*batchv1.Job, error) {
	generatorConfig := function.Spec.Function.GeneratorConfig
	if generatorConfig == nil {
		generatorConfig = function.Status.GeneratorConfig
	}

	if generatorConfig.Image == "" {
		return nil, fmt.Errorf("incorrect generator config state! GeneratorConfig empty: %s/%s", function.Namespace, function.Name)
	}

	// Create Job name
	jobName := fmt.Sprintf("%s-codegen-%d", function.Name, function.Generation)

	list := batchv1.JobList{}
	err := c.List(ctx, &list, client.InNamespace(function.Namespace), client.MatchingFields{
		"metadata.name": jobName,
	})
	if err != nil {
		return nil, err
	}

	if len(list.Items) > 0 {
		return &list.Items[0], nil
	}

	functionString := string(marshalFunctionSpec(function))

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
							Name:    "git-checkout",
							Image:   generatorConfig.Git.Image,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								`
cd /shared

# 1. Setup identity and SSH Signing
mkdir -p ~/.ssh

gpg --import /var/secrets/gpg/gpg.key &2>/dev/null
(echo "4"; echo "y"; echo "save") | gpg --command-fd 0 --edit-key $GPG_KEY_ID trust
export GPG_TTY=$(tty)

git config --global user.email "${COMMITTER_EMAIL}"
git config --global user.name "${COMMITTER_NAME}"
git config --global user.signingkey $GPG_KEY_ID
git config --global commit.gpgsign true

# 2. Clone the repo (using a token for auth)
git clone https://x-access-token:${GIT_TOKEN}@${GIT_HOST}/${GIT_ORG}/${GIT_REPO} .

# 3. Sanitize variables to create a safe branch name
# Example: Name="my-app", BasePath="/api/v1" -> my-app-api-v1
SAFE_PATH=$(echo "${FUNCTION_BASEPATH}" | sed 's/\//-/g' | sed 's/^-//')
BRANCH_NAME="gen/${NAMESPACE}/${FUNCTION_NAME}-${SAFE_PATH}"

# 4. Check if branch exists on remote, then switch or create
if git ls-remote --heads origin ${BRANCH_NAME} | grep -q ${BRANCH_NAME}; then
  git fetch origin ${BRANCH_NAME}
  git checkout ${BRANCH_NAME}
else
  git checkout -b ${BRANCH_NAME}
fi

# 5. Create the subdirectory if it doesn't exist
# We target the specific subdirectory for this function
TARGET_DIR="./functions/${NAMESPACE}/${FUNCTION_NAME}-${SAFE_PATH}"
mkdir -p $TARGET_DIR

# 6. Ignore the .env files
if ! grep -q ".env" .gitignore; then
  echo ".env" >> .gitignore
  echo "Added .env to .gitignore"
fi
echo "TARGET_DIR=${TARGET_DIR}" > .env
								`,
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: generatorConfig.Git.RepoSecretRef,
									},
								},
							},
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
									Name:  "FUNCTION_BASEPATH",
									Value: function.Spec.API.BasePath,
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
									Name: "GIT_TOKEN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key:                  "token",
											LocalObjectReference: generatorConfig.Git.RepoSecretRef,
										},
									},
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
									Name: "GPG_KEY_ID",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key:                  "gpg.key.id",
											LocalObjectReference: generatorConfig.Git.RepoSecretRef,
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-data",
									MountPath: "/shared",
								},
								{
									Name:      "gpg-key",
									MountPath: "/var/secrets/gpg",
									ReadOnly:  true,
								},
							},
						},
						{
							Name:    "generate-code",
							Image:   generatorConfig.Image,
							Command: generatorConfig.Command,
							Args:    generatorConfig.Args,
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
									Value: functionString,
								},
								{
									Name:  "WORKING_DIRECTORY",
									Value: "/shared",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-data",
									MountPath: "/shared",
								},
							},
						},
						{
							Name:    "git-push",
							Image:   generatorConfig.Git.Image,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{`
cd /shared
source .env

# 5. Commit and Push
git add $TARGET_DIR
if git diff-index --quiet HEAD; then
  echo "No changes detected for this iteration."
else
  git commit -S -m "Update function: ${FUNC_NAME} at ${BASE_PATH}"
  git push origin $BRANCH_NAME
fi

# Write details to .env
echo "FUNCTION_NAME=${FUNCTION_NAME}" >> .env
echo "BASE_PATH=${BASE_PATH}" >> .env
echo "BRANCH_NAME=${BRANCH_NAME}" >> .env
echo "REPOSITORY=$(git remote get-url --push origin)" >> .env

# Payload containing the metadata needed by the Dispatcher
#PAYLOAD=$(cat <<EOF
#{
#  "function_name": "${FUNC_NAME}",
#  "base_path": "${BASE_PATH}",
#  "branch_name": "${BRANCH_NAME}",
#  "repository": "org/my-repo",
#  "provider": "github"
#}
#EOF
#)

# Call the Dispatcher Service (internal cluster DNS)
#curl -X POST http://git-dispatcher.tools.svc.cluster.local/create-pr \
#     -H "Content-Type: application/json" \
#     -H "X-Dispatcher-Token: ${INTERNAL_AUTH_TOKEN}" \
#     -d "$PAYLOAD"
`},
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
									Name:  "FUNCTION_BASEPATH",
									Value: function.Spec.API.BasePath,
								},
								{
									Name:  "COMMITTER_EMAIL",
									Value: generatorConfig.Git.CommitterEmail,
								},
								{
									Name:  "COMMITTER_NAME",
									Value: generatorConfig.Git.CommitterName,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-data",
									MountPath: "/shared",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "update-status",
							Image: "busybox:latest",

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-data",
									MountPath: "/shared",
								},
							},

							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								`
cd /shared
source .env
curl -X PATCH -H "Content-Type: application/json-patch+json" --data '{
	"status":{
		"state":"StubGenerated"
	}
}' http://api-server/v1/namespaces/${NAMESPACE}/kdexfunctions/${FUNCTION_NAME}
`,
							},

							// Optional: Set resource limits
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("50Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("100Mi"),
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: sa,
					Volumes: []corev1.Volume{
						{
							Name: "shared-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gpg-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: generatorConfig.Git.RepoSecretRef.Name,
									Items: []corev1.KeyToPath{
										{
											Key:  "gpg.key",
											Path: "gpg.key",
										},
									},
								},
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

func marshalFunctionSpec(fn *kdexv1alpha1.KDexFunction) []byte {
	bytes, _ := json.MarshalIndent(fn, "", "  ")
	return bytes
}
