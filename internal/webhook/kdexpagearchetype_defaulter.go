package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagearchetype,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagearchetypes,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagearchetype.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpagearchetype,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagearchetypes,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpagearchetype.kdex.dev,admissionReviewVersions=v1

type KDexPageArchetypeDefaulter struct {
}

func (a *KDexPageArchetypeDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexPageArchetypeSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexPageArchetype:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageArchetype:
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.DefaultFooterRef != nil && spec.DefaultFooterRef.Kind == "" {
		spec.DefaultFooterRef.Kind = "KDexPageFooter"
	}

	if spec.DefaultHeaderRef != nil && spec.DefaultHeaderRef.Kind == "" {
		spec.DefaultHeaderRef.Kind = "KDexPageHeader"
	}

	if spec.DefaultMainNavigationRef != nil && spec.DefaultMainNavigationRef.Kind == "" {
		spec.DefaultMainNavigationRef.Kind = KDexPageNavigation
	}

	for _, v := range spec.ExtraNavigations {
		if v.Kind == "" {
			v.Kind = KDexPageNavigation
		}
	}

	if spec.ScriptLibraryRef != nil && spec.ScriptLibraryRef.Kind == "" {
		spec.ScriptLibraryRef.Kind = KDexScriptLibrary
	}

	return nil
}
