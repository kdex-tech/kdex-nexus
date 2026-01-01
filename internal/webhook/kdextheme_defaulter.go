package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdextheme,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexthemes,verbs=create;update,versions=v1alpha1,name=mutate.kdextheme.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclustertheme,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterthemes,verbs=create;update,versions=v1alpha1,name=mutate.kdexclustertheme.kdex.dev,admissionReviewVersions=v1

type KDexThemeDefaulter struct {
}

func (a *KDexThemeDefaulter) Default(ctx context.Context, ro runtime.Object) error {
	var spec *kdexv1alpha1.KDexThemeSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexTheme:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterTheme:
		spec = &t.Spec
	default:
		return fmt.Errorf("unsupported type: %T", t)
	}

	if spec.ScriptLibraryRef != nil && spec.ScriptLibraryRef.Kind == "" {
		spec.ScriptLibraryRef.Kind = KDexScriptLibrary
	}

	spec.IngressPath = "/_theme"

	BackendDefaults(&spec.Backend)

	return nil
}
