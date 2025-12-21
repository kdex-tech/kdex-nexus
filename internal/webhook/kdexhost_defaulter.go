package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexhost,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexhosts,verbs=create;update,versions=v1alpha1,name=mutate.kdexhost.kdex.dev,admissionReviewVersions=v1

type KDexHostDefaulter struct {
}

func (a *KDexHostDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexHostSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexHost:
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.DefaultLang == "" {
		spec.DefaultLang = "en"
	}

	if spec.ModulePolicy == "" {
		spec.ModulePolicy = kdexv1alpha1.StrictModulePolicy
	}

	if spec.Routing.Strategy == "" {
		spec.Routing.Strategy = kdexv1alpha1.IngressRoutingStrategy
	}

	spec.IngressPath = "/_host"

	return nil
}
