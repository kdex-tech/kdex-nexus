package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdextheme,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexthemes,verbs=create;update,versions=v1alpha1,name=validate.kdextheme.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclustertheme,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterthemes,verbs=create;update,versions=v1alpha1,name=validate.kdexclustertheme.kdex.dev,admissionReviewVersions=v1

type KDexThemeValidator struct {
}

var _ admission.CustomValidator = &KDexThemeValidator{}

func (v *KDexThemeValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *KDexThemeValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *KDexThemeValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *KDexThemeValidator) validate(ctx context.Context, o runtime.Object) (admission.Warnings, error) {
	var spec *kdexv1alpha1.KDexThemeSpec

	if obj, ok := o.(*kdexv1alpha1.KDexTheme); ok {
		spec = &obj.Spec
	} else if obj, ok := o.(*kdexv1alpha1.KDexClusterTheme); ok {
		spec = &obj.Spec
	} else {
		return nil, fmt.Errorf("expected KDexTheme|KDexClusterTheme but got %T", obj)
	}

	if err := validation.ValidateAssets(spec.Assets); err != nil {
		return nil, err
	}

	// Validate ResourceProvider
	if err := validation.ValidateResourceProvider(spec); err != nil {
		return nil, err
	}

	return nil, nil
}
