package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexutilitypage,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexutilitypages,verbs=create;update,versions=v1alpha1,name=mutate.kdexutilitypage.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterutilitypage,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterutilitypages,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterutilitypage.kdex.dev,admissionReviewVersions=v1

type KDexUtilityPageDefaulter struct {
}

func (a *KDexUtilityPageDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexUtilityPageSpec
	clustered := false

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexUtilityPage:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterUtilityPage:
		clustered = true
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	for _, entry := range spec.ContentEntries {
		if entry.AppRef != nil {
			if clustered {
				entry.AppRef.Kind = KDexClusterApp
			} else {
				entry.AppRef.Kind = KDexApp
			}
		}
	}

	if spec.OverrideFooterRef != nil && spec.OverrideFooterRef.Kind == "" {
		if clustered {
			spec.OverrideFooterRef.Kind = KDexClusterPageFooter
		} else {
			spec.OverrideFooterRef.Kind = KDexPageFooter
		}
	}

	if spec.OverrideHeaderRef != nil && spec.OverrideHeaderRef.Kind == "" {
		if clustered {
			spec.OverrideHeaderRef.Kind = KDexClusterPageHeader
		} else {
			spec.OverrideHeaderRef.Kind = KDexPageHeader
		}
	}

	if spec.OverrideMainNavigationRef != nil && spec.OverrideMainNavigationRef.Kind == "" {
		if clustered {
			spec.OverrideMainNavigationRef.Kind = KDexClusterPageNavigation
		} else {
			spec.OverrideMainNavigationRef.Kind = KDexPageNavigation
		}
	}

	if spec.PageArchetypeRef.Kind == "" {
		if clustered {
			spec.PageArchetypeRef.Kind = KDexClusterPageArchetype
		} else {
			spec.PageArchetypeRef.Kind = KDexPageArchetype
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
