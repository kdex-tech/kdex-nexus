package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagefooter,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagefooters,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagefooter.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpagefooter,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagefooters,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpagefooter.kdex.dev,admissionReviewVersions=v1

type KDexPageFooterDefaulter struct {
}

func (a *KDexPageFooterDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexPageFooterSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexPageFooter:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageFooter:
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.ScriptLibraryRef != nil && spec.ScriptLibraryRef.Kind == "" {
		spec.ScriptLibraryRef.Kind = "KDexScriptLibrary"
	}

	return nil
}
