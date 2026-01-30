package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexscriptlibrary,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexscriptlibraries,verbs=create;update,versions=v1alpha1,name=mutate.kdexscriptlibrary.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterscriptlibrary,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterscriptlibraries,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterscriptlibrary.kdex.dev,admissionReviewVersions=v1

type KDexScriptLibraryDefaulter[T runtime.Object] struct {
}

func (a *KDexScriptLibraryDefaulter[T]) Default(ctx context.Context, obj T) error {
	var name string
	var spec *kdexv1alpha1.KDexScriptLibrarySpec

	switch t := any(obj).(type) {
	case *kdexv1alpha1.KDexScriptLibrary:
		name = t.Name
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterScriptLibrary:
		name = t.Name
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	spec.IngressPath = "/-/s/" + name

	BackendDefaults(&spec.Backend)

	return nil
}
