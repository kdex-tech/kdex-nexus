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
								`set -e

cd /shared

# 1. Clone the repo
# We use the 'admin' username and the token we generated earlier
git clone http://${GIT_USER}:${GIT_TOKEN}@${GIT_HOST}/${GIT_ORG}/${GIT_REPO} .

# 2. Basic Git Identity (No Signing)
git config user.email "${COMMITTER_EMAIL}"
git config user.name "${COMMITTER_NAME}"
# Explicitly disable signing to avoid errors if the runner has global defaults
git config commit.gpgsign false

# 3. Sanitize variables to create a safe branch name
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
mkdir -p "${TARGET_DIR}"

# 6. Ignore the .env files
grep -q ".env" .gitignore 2>/dev/null || echo ".env" >> .gitignore
echo "TARGET_DIR=${TARGET_DIR}" > .env
echo "BRANCH_NAME=${BRANCH_NAME}" >> .env
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
									Name: "GIT_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key:                  "user",
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
									Value: string(functionString),
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
							Args: []string{`set -e

cd /shared
source .env

# 1. Commit and Push
git add $TARGET_DIR
if git diff-index --quiet HEAD; then
  echo "No changes detected for this iteration."
else
  git commit -m "Update function: ${FUNCTION_NAME} at ${FUNCTION_BASEPATH}"
  git push origin $BRANCH_NAME
fi

# Write details to .env
echo "FUNCTION_NAME=${FUNCTION_NAME}" >> .env
echo "BASE_PATH=${FUNCTION_BASEPATH}" >> .env
echo "BRANCH_NAME=${BRANCH_NAME}" >> .env
echo "REPOSITORY=$(git remote get-url --push origin)" >> .env
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
							Name:    "update-status",
							Image:   generatorConfig.Git.Image,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								`set -e

cd /shared
source .env

APISERVER="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}"
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
CACERT="/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

# Remove any authority from the REPOSITORY URL
REPOSITORY=$(gurl +%S%H%p ${REPOSITORY})

DATA=$(cat <<EOF
{
	"status": {
		"state": "StubGenerated",
		"stubDetails": {
			"sourcePath": "${REPOSITORY}/src/branch/${BRANCH_NAME}"
		},
		"detail": "Source: ${REPOSITORY}/src/branch/${BRANCH_NAME}"
	}
}
EOF
)

RESPONSE=$(\
	curl ${APISERVER}/apis/kdex.dev/v1alpha1/namespaces/${NAMESPACE}/kdexfunctions/${FUNCTION_NAME}/status \
		-s -w "%{http_code}" \
		--cacert $CACERT \
		-X PATCH \
		-H "Content-Type: application/merge-patch+json" \
		-H "Authorization: Bearer ${TOKEN}" \
		--data "${DATA}")

HTTP_CODE="${RESPONSE:${#RESPONSE}-3}"
CONTENT="${RESPONSE:0:${#RESPONSE}-3}"

echo "Status Code: $HTTP_CODE"
echo "Response Body: $CONTENT"

if [ "$HTTP_CODE" -eq 200 ]; then
	echo "Success!"
else
	echo "Patch failed with error."
	exit 1
fi
`,
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
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-data",
									MountPath: "/shared",
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
