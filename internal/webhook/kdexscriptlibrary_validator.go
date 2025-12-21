package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexscriptlibrary,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexscriptlibraries,verbs=create;update,versions=v1alpha1,name=validate.kdexscriptlibrary.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclusterscriptlibrary,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterscriptlibraries,verbs=create;update,versions=v1alpha1,name=validate.kdexclusterscriptlibrary.kdex.dev,admissionReviewVersions=v1

type KDexScriptLibraryValidator struct {
}

var _ admission.CustomValidator = &KDexScriptLibraryValidator{}

func (v *KDexScriptLibraryValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *KDexScriptLibraryValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *KDexScriptLibraryValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *KDexScriptLibraryValidator) validate(ctx context.Context, ro runtime.Object) (admission.Warnings, error) {
	var spec *kdexv1alpha1.KDexScriptLibrarySpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexScriptLibrary:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterScriptLibrary:
		spec = &t.Spec
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	if err := validation.ValidateScriptLibrary(spec); err != nil {
		return nil, err
	}

	// Validate ResourceProvider
	if err := validation.ValidateResourceProvider(spec); err != nil {
		return nil, err
	}

	return nil, nil
}
