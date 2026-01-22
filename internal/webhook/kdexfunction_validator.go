package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-kdex-dev-v1alpha1-kdexfunction,mutating=false,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexfunctions,verbs=create;update,versions=v1alpha1,name=validate.kdexfunction.kdex.dev,admissionReviewVersions=v1

type KDexFunctionValidator[T runtime.Object] struct {
}

var _ admission.Validator[*kdexv1alpha1.KDexFunction] = &KDexFunctionValidator[*kdexv1alpha1.KDexFunction]{}

func (v *KDexFunctionValidator[T]) ValidateCreate(ctx context.Context, obj T) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *KDexFunctionValidator[T]) ValidateUpdate(ctx context.Context, oldObj, newObj T) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *KDexFunctionValidator[T]) ValidateDelete(ctx context.Context, obj T) (admission.Warnings, error) {
	return nil, nil
}

func (v *KDexFunctionValidator[T]) validate(_ context.Context, obj T) (admission.Warnings, error) {
	var function *kdexv1alpha1.KDexFunction

	switch t := any(obj).(type) {
	case *kdexv1alpha1.KDexFunction:
		function = t
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	spec := &function.Spec

	re := spec.API.BasePathRegex()

	if !re.MatchString(spec.API.BasePath) {
		return nil, fmt.Errorf(".spec.api.basePath %s does not match %s", spec.API.BasePath, re.String())
	}

	re = spec.API.ItemPathRegex()

	for curPath := range spec.API.Paths {
		if !re.MatchString(curPath) {
			return nil, fmt.Errorf(".spec.api.paths[%s] does not match %s", curPath, re.String())
		}
	}

	return nil, nil
}
