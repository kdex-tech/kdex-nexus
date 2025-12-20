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

func (v *KDexHostValidator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	host, ok := obj.(*kdexv1alpha1.KDexHost)
	if !ok {
		return nil, fmt.Errorf("expected KDexHost but got %T", obj)
	}

	if host.Spec.BrandName == "" {
		return nil, fmt.Errorf(`spec.brandName: Invalid value: ""`)
	}

	if err := validation.ValidateAssets(host.Spec.Assets); err != nil {
		return nil, err
	}

	// Validate ResourceProvider
	if err := validation.ValidateResourceProvider(&host.Spec); err != nil {
		return nil, err
	}

	return nil, nil
}
