package webhook

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexfunction,mutating=true,failurePolicy=Ignore,sideEffects=None,groups=kdex.dev,resources=kdexfunctions;kdexfunctions/status,verbs=create;update,versions=v1alpha1,name=mutate.kdexfunction.kdex.dev,admissionReviewVersions=v1

type KDexFunctionDefaulter[T runtime.Object] struct {
}

var _ admission.Defaulter[*kdexv1alpha1.KDexFunction] = &KDexFunctionDefaulter[*kdexv1alpha1.KDexFunction]{}

func (a *KDexFunctionDefaulter[T]) Default(ctx context.Context, obj T) error {
	// var function *kdexv1alpha1.KDexFunction

	// switch t := any(obj).(type) {
	// case *kdexv1alpha1.KDexFunction:
	// 	function = t
	// default:
	// 	return fmt.Errorf("unsupported type: %T", t)
	// }

	// if function.Status.State == "" {
	// 	function.Status.State = kdexv1alpha1.KDexFunctionStatePending
	// }

	return nil
}
