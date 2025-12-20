package webhook

import (
	"context"
	"fmt"
	"strings"

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

func (v *KDexAppValidator) validate(ctx context.Context, o runtime.Object) (admission.Warnings, error) {
	var spec *kdexv1alpha1.KDexAppSpec

	if obj, ok := o.(*kdexv1alpha1.KDexApp); ok {
		spec = &obj.Spec
	} else if obj, ok := o.(*kdexv1alpha1.KDexClusterApp); ok {
		spec = &obj.Spec
	} else {
		return nil, fmt.Errorf("expected KDexApp|KDexClusterApp but got %T", obj)
	}

	// Validate PackageReference name
	if !strings.HasPrefix(spec.PackageReference.Name, "@") || !strings.Contains(spec.PackageReference.Name, "/") {
		return nil, fmt.Errorf("invalid package name, must be scoped with @scope/name: %s", spec.PackageReference.Name)
	}

	// Validate ResourceProvider
	if err := validation.ValidateResourceProvider(spec); err != nil {
		return nil, err
	}

	return nil, nil
}
