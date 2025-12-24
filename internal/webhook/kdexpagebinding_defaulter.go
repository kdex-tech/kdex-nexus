package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpagebinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagebindings,verbs=create;update,versions=v1alpha1,name=mutate.kdexpagebinding.kdex.dev,admissionReviewVersions=v1

type KDexPageBindingDefaulter struct {
}

func (a *KDexPageBindingDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexPageBindingSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexPageBinding:
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	for _, entry := range spec.ContentEntries {
		if entry.AppRef != nil {
			if entry.AppRef.Kind == "" {
				entry.AppRef.Kind = "KDexApp"
			}
		}
	}

	if spec.OverrideFooterRef != nil && spec.OverrideFooterRef.Kind == "" {
		spec.OverrideFooterRef.Kind = "KDexPageFooter"
	}

	if spec.OverrideHeaderRef != nil && spec.OverrideHeaderRef.Kind == "" {
		spec.OverrideHeaderRef.Kind = "KDexPageHeader"
	}

	if spec.OverrideMainNavigationRef != nil && spec.OverrideMainNavigationRef.Kind == "" {
		spec.OverrideMainNavigationRef.Kind = KDexPageNavigation
	}

	if spec.PageArchetypeRef.Kind == "" {
		spec.PageArchetypeRef.Kind = "KDexPageArchetype"
	}

	if spec.ScriptLibraryRef != nil && spec.ScriptLibraryRef.Kind == "" {
		spec.ScriptLibraryRef.Kind = KDexScriptLibrary
	}

	return nil
}
