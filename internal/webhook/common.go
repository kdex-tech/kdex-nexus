package webhook

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"kdex.dev/crds/api/v1alpha1"
)

func BackendDefaults(backend *v1alpha1.Backend) {
	if backend.ServerImage != "" && backend.ServerImagePullPolicy == "" {
		if strings.HasSuffix(backend.ServerImage, ":latest") {
			backend.ServerImagePullPolicy = corev1.PullAlways
		} else {
			backend.ServerImagePullPolicy = corev1.PullIfNotPresent
		}
	}

	if backend.StaticImage != "" && backend.StaticImagePullPolicy == "" {
		if strings.HasSuffix(backend.StaticImage, ":latest") {
			backend.StaticImagePullPolicy = corev1.PullAlways
		} else {
			backend.StaticImagePullPolicy = corev1.PullIfNotPresent
		}
	}
}
