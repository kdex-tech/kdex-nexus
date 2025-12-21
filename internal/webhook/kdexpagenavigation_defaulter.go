package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagenavigation,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagenavigations,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagenavigation.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpagenavigation,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagenavigations,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpagenavigation.kdex.dev,admissionReviewVersions=v1

type KDexPageNavigationDefaulter struct {
}

func (a *KDexPageNavigationDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexPageNavigationSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexPageNavigation:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageNavigation:
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.ScriptLibraryRef != nil && spec.ScriptLibraryRef.Kind == "" {
		spec.ScriptLibraryRef.Kind = "KDexScriptLibrary"
	}

	return nil
}
