package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexapps,verbs=create;update,versions=v1alpha1,name=mutate.kdexapp.kdex.dev,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexclusterapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexclusterapps,verbs=create;update,versions=v1alpha1,name=mutate.kdexclusterapp.kdex.dev,admissionReviewVersions=v1

type KDexAppDefaulter struct {
}

func (a *KDexAppDefaulter) Default(ctx context.Context, o runtime.Object) error {
	var spec *kdexv1alpha1.KDexAppSpec

	if obj, ok := o.(*kdexv1alpha1.KDexApp); ok {
		spec = &obj.Spec
	} else if obj, ok := o.(*kdexv1alpha1.KDexClusterApp); ok {
		spec = &obj.Spec
	} else {
		return fmt.Errorf("expected KDexApp|KDexClusterApp but got %T", obj)
	}

	if spec.WebServer.IngressPath == "" {
		spec.WebServer.IngressPath = "/" + o.(client.Object).GetName()
	}

	return nil
}
