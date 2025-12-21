package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexhost,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexhosts,verbs=create;update,versions=v1alpha1,name=validate.kdexhost.kdex.dev,admissionReviewVersions=v1

type KDexHostValidator struct {
}

var _ admission.CustomValidator = &KDexHostValidator{}

func (v *KDexHostValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *KDexHostValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *KDexHostValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *KDexHostValidator) validate(ctx context.Context, ro runtime.Object) (admission.Warnings, error) {
	var spec *kdexv1alpha1.KDexHostSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexHost:
		spec = &t.Spec
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	if spec.BrandName == "" {
		return nil, fmt.Errorf(`spec.brandName: Invalid value: ""`)
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
