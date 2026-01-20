package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexpageheader,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpageheaders,verbs=create;update,versions=v1alpha1,name=mutate.kdexpageheader.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterpageheader,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpageheaders,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterpageheader.kdex.dev,admissionReviewVersions=v1

type KDexPageHeaderDefaulter[T runtime.Object] struct {
}

func (a *KDexPageHeaderDefaulter[T]) Default(ctx context.Context, obj T) error {
	var spec *kdexv1alpha1.KDexPageHeaderSpec
	clustered := false

	switch t := any(obj).(type) {
	case *kdexv1alpha1.KDexPageHeader:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterPageHeader:
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
