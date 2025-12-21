package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexpagearchetype,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagearchetypes,verbs=create;update,versions=v1alpha1,name=validation.kdexpagearchetype.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexpagefooter,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagefooters,verbs=create;update,versions=v1alpha1,name=validation.kdexpagefooter.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexpageheader,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpageheaders,verbs=create;update,versions=v1alpha1,name=validation.kdexpageheader.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexpagenavigation,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagenavigations,verbs=create;update,versions=v1alpha1,name=validation.kdexpagenavigation.kdex.dev,admissionReviewVersions=v1

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclusterpagearchetype,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagearchetypes,verbs=create;update,versions=v1alpha1,name=validation.kdexclusterpagearchetype.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclusterpagefooter,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagefooters,verbs=create;update,versions=v1alpha1,name=validation.kdexclusterpagefooter.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclusterpageheader,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpageheaders,verbs=create;update,versions=v1alpha1,name=validation.kdexclusterpageheader.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexclusterpagenavigation,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterpagenavigations,verbs=create;update,versions=v1alpha1,name=validation.kdexclusterpagenavigation.kdex.dev,admissionReviewVersions=v1

type PageContentValidator struct{}

var _ admission.CustomValidator = &PageContentValidator{}

func (v *PageContentValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *PageContentValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *PageContentValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *PageContentValidator) validate(ctx context.Context, ro runtime.Object) (admission.Warnings, error) {
	var content string
	var name string

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexPageArchetype:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexPageFooter:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexPageHeader:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexPageNavigation:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexClusterPageArchetype:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexClusterPageFooter:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexClusterPageHeader:
		content = t.Spec.Content
		name = t.Name
	case *kdexv1alpha1.KDexClusterPageNavigation:
		content = t.Spec.Content
		name = t.Name
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	if err := render.ValidateContent(name, content); err != nil {
		return nil, fmt.Errorf("invalid go template in spec.content: %w", err)
	}

	return nil, nil
}
