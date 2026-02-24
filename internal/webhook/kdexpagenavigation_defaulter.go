package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagenavigation,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexpagenavigations,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagenavigation.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpagenavigation,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagenavigations,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpagenavigation.kdex.dev,admissionReviewVersions=v1

type KDexPageNavigationDefaulter[T runtime.Object] struct {
}

func (a *KDexPageNavigationDefaulter[T]) Default(ctx context.Context, obj T) error {
	var spec *kdexv1alpha1.KDexPageNavigationSpec
	clustered := false

	switch t := any(obj).(type) {
	case *kdexv1alpha1.KDexPageNavigation:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageNavigation:
		clustered = true
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.ScriptLibraryRef != nil && spec.ScriptLibraryRef.Kind == "" {
		if clustered {
			spec.ScriptLibraryRef.Kind = KDexClusterScriptLibrary
		} else {
			spec.ScriptLibraryRef.Kind = KDexScriptLibrary
		}
	}

	return nil
}
