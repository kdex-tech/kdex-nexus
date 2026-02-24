package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagefooter,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexpagefooters,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagefooter.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpagefooter,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagefooters,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpagefooter.kdex.dev,admissionReviewVersions=v1

type KDexPageFooterDefaulter[T runtime.Object] struct {
}

func (a *KDexPageFooterDefaulter[T]) Default(ctx context.Context, obj T) error {
	var spec *kdexv1alpha1.KDexPageFooterSpec
	clustered := false

	switch t := any(obj).(type) {
	case *kdexv1alpha1.KDexPageFooter:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageFooter:
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
