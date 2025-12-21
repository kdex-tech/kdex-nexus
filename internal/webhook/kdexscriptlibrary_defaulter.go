package webhook

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexscriptlibrary,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexscriptlibraries,verbs=create;update,versions=v1alpha1,name=mutate.kdexscriptlibrary.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterscriptlibrary,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterscriptlibraries,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterscriptlibrary.kdex.dev,admissionReviewVersions=v1

type KDexScriptLibraryDefaulter struct {
}

func (a *KDexScriptLibraryDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var name string
	var spec *kdexv1alpha1.KDexScriptLibrarySpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexScriptLibrary:
		name = t.Name
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterScriptLibrary:
		name = t.Name
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	spec.IngressPath = "/_s/" + name

	if spec.ServerImage != "" && spec.ServerImagePullPolicy == "" {
		if strings.HasSuffix(spec.ServerImage, ":latest") {
			spec.ServerImagePullPolicy = v1.PullAlways
		} else {
			spec.ServerImagePullPolicy = v1.PullIfNotPresent
		}
	}

	if spec.StaticImage != "" && spec.StaticImagePullPolicy == "" {
		if strings.HasSuffix(spec.StaticImage, ":latest") {
			spec.StaticImagePullPolicy = v1.PullAlways
		} else {
			spec.StaticImagePullPolicy = v1.PullIfNotPresent
		}
	}

	return nil
}
