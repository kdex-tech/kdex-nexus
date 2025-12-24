package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexpagebinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexpagebindings,verbs=create;update,versions=v1alpha1,name=validate.kdexpagebinding.kdex.dev,admissionReviewVersions=v1

type KDexPageBindingValidator struct {
}

var _ admission.CustomValidator = &KDexPageBindingValidator{}

func (v *KDexPageBindingValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *KDexPageBindingValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *KDexPageBindingValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *KDexPageBindingValidator) validate(_ context.Context, ro runtime.Object) (admission.Warnings, error) {
	var spec *kdexv1alpha1.KDexPageBindingSpec

	switch t := ro.(type) {
	case *kdexv1alpha1.KDexPageBinding:
		spec = &t.Spec
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	if spec.BasePath == "/" && spec.ParentPageRef != nil {
		return nil, fmt.Errorf("pagebinding with basePath '/' must not specify a parent page binding")
	}

	for idx, entry := range spec.ContentEntries {
		if entry.RawHTML != "" {
			if err := render.ValidateContent(entry.Slot, entry.RawHTML); err != nil {
				return nil, fmt.Errorf("invalid go template in spec.contentEntries[%d].rawHTML: %w", idx, err)
			}
		}
		if entry.AppRef != nil && entry.AppRef.Name == "" {
			return nil, fmt.Errorf("spec.contentEntries[%d].appRef.name is required", idx)
		}
	}

	return nil, nil
}
