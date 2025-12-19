package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
)

// +kubebuilder:webhook:path=/mutate-kdex-dev-v1alpha1-kdexapps,mutating=true,failurePolicy=fail,sideEffects=None,groups=kdex.dev,resources=kdexapps,verbs=create;update,versions=v1alpha1,name=kdexapps.kdex.dev,admissionReviewVersions=v1

type KDexAppDefaulter struct {
	Client        client.Client
	Configuration configuration.NexusConfiguration
	decoder       admission.Decoder
}

// InjectDecoder injects the decoder.
func (a *KDexAppDefaulter) InjectDecoder(d admission.Decoder) error {
	a.decoder = d
	return nil
}

func (a *KDexAppDefaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	obj := &kdexv1alpha1.KDexApp{}

	err := a.decoder.Decode(req, obj)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if obj.Spec.WebServer.IngressPath == "" {
		obj.Spec.WebServer.IngressPath = "/" + obj.GetName()
	}

	marshaledObj, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledObj)
}
