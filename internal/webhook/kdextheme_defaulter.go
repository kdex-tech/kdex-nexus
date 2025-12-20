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

func (a *KDexThemeDefaulter) Default(ctx context.Context, o runtime.Object) error {
	var spec *kdexv1alpha1.KDexThemeSpec

	if obj, ok := o.(*kdexv1alpha1.KDexTheme); ok {
		spec = &obj.Spec
	} else if obj, ok := o.(*kdexv1alpha1.KDexClusterTheme); ok {
		spec = &obj.Spec
	} else {
		return fmt.Errorf("expected KDexTheme|KDexClusterTheme but got %T", obj)
	}

	if spec.WebServer.IngressPath == "" {
		spec.WebServer.IngressPath = "/theme"
	}

	return nil
}
