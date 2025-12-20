package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexhost,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexhosts,verbs=create;update,versions=v1alpha1,name=mutate.kdexhost.kdex.dev,admissionReviewVersions=v1

type KDexHostDefaulter struct {
}

func (a *KDexHostDefaulter) Default(ctx context.Context, o runtime.Object) error {
	obj, ok := o.(*kdexv1alpha1.KDexHost)

	if !ok {
		return fmt.Errorf("expected KDexHost but got %T", obj)
	}

	if obj.Spec.DefaultLang == "" {
		obj.Spec.DefaultLang = "en"
	}

	if obj.Spec.ModulePolicy == "" {
		obj.Spec.ModulePolicy = kdexv1alpha1.StrictModulePolicy
	}

	if obj.Spec.WebServer.IngressPath == "" {
		obj.Spec.WebServer.IngressPath = "/static"
	}

	return nil
}
