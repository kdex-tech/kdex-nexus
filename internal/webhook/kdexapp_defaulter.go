package webhook

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexapps,verbs=create;update,versions=v1alpha1,name=mutate.kdexapp.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterapps,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterapp.kdex.dev,admissionReviewVersions=v1

type KDexAppDefaulter struct {
}

func (a *KDexAppDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var name string
	var spec *kdexv1alpha1.KDexAppSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexApp:
		name = t.Name
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterApp:
		name = t.Name
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	spec.IngressPath = "/_a/" + name

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
