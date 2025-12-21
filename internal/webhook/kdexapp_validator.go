package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexapps,verbs=create;update,versions=v1alpha1,name=validate.kdexapp.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclusterapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterapps,verbs=create;update,versions=v1alpha1,name=validate.kdexclusterapp.kdex.dev,admissionReviewVersions=v1

type KDexAppValidator struct {
}

var _ admission.CustomValidator = &KDexAppValidator{}

func (v *KDexAppValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *KDexAppValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *KDexAppValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *KDexAppValidator) validate(_ context.Context, ro runtime.Object) (admission.Warnings, error) {
	var spec *kdexv1alpha1.KDexAppSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexApp:
		spec = &t.Spec
	case *kdexv1alpha1.KDexClusterApp:
		spec = &t.Spec
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	// apply the same logic as KDexScriptLibrary
	sl := &kdexv1alpha1.KDexScriptLibrarySpec{
		Backend:          spec.Backend,
		PackageReference: &spec.PackageReference,
		Scripts:          spec.Scripts,
	}

	if err := validation.ValidateScriptLibrary(sl); err != nil {
		return nil, err
	}

	// Validate ResourceProvider
	if err := validation.ValidateResourceProvider(spec); err != nil {
		return nil, err
	}

	return nil, nil
}
