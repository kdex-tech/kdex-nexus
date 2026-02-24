package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagearchetype,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexpagearchetypes,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagearchetype.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpagearchetype,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagearchetypes,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpagearchetype.kdex.dev,admissionReviewVersions=v1

type KDexPageArchetypeDefaulter[T runtime.Object] struct {
}

func (a *KDexPageArchetypeDefaulter[T]) Default(ctx context.Context, obj T) error {
	var spec *kdexv1alpha1.KDexPageArchetypeSpec
	clustered := false

	switch t := any(obj).(type) {
	case *kdexv1alpha1.KDexPageArchetype:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageArchetype:
		clustered = true
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.DefaultFooterRef != nil && spec.DefaultFooterRef.Kind == "" {
		if clustered {
			spec.DefaultFooterRef.Kind = KDexClusterPageFooter
		} else {
			spec.DefaultFooterRef.Kind = KDexPageFooter
		}
	}

	if spec.DefaultHeaderRef != nil && spec.DefaultHeaderRef.Kind == "" {
		if clustered {
			spec.DefaultHeaderRef.Kind = KDexClusterPageHeader
		} else {
			spec.DefaultHeaderRef.Kind = KDexPageHeader
		}
	}

	for _, v := range spec.DefaultNavigationRefs {
		if v.Kind == "" {
			if clustered {
				v.Kind = KDexClusterPageNavigation
			} else {
				v.Kind = KDexPageNavigation
			}
		}
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
