package build

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	"kdex.dev/nexus/internal"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Builder struct {
	client.Client
	Scheme        *runtime.Scheme
	Configuration configuration.NexusConfiguration
	Source        kdexv1alpha1.Source
}

func (b *Builder) GetOrCreateKPackImage(ctx context.Context, function *kdexv1alpha1.KDexFunction) (*unstructured.Unstructured, error) {
	kImageName := fmt.Sprintf("%s-%s", function.Spec.HostRef.Name, function.Name)

	// an unstructured object is used to avoid a dependency on the kpack.io API group
	kImage := &unstructured.Unstructured{}
	kImage.SetGroupVersionKind(internal.KPackImageGVK)
	err := b.Get(ctx, client.ObjectKey{Namespace: function.Namespace, Name: kImageName}, kImage)
	if err == nil {
		return kImage, nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	kImage = &unstructured.Unstructured{}
	kImage.SetGroupVersionKind(internal.KPackImageGVK)
	kImage.SetNamespace(function.Namespace)
	kImage.SetName(kImageName)

	op, err := ctrl.CreateOrPatch(ctx, b.Client, kImage, func() error {
		kImage.SetLabels(map[string]string{
			"app":           "builder",
			"function":      function.Name,
			"kdex.dev/host": function.Spec.HostRef.Name,
		})
		kImage.Object = map[string]any{
			"spec": map[string]any{
				"build": map[string]any{
					"env": []any{
						map[string]any{
							"name":  "BP_LOG_LEVEL",
							"value": "DEBUG",
						},
					},
				},
				"builder": map[string]any{
					"name": "tiny-microservice-builder",
					"kind": "ClusterBuilder",
				},
				"imageTaggingStrategy": "BuildNumber",
				"source": map[string]any{
					"git": map[string]any{
						"url":      b.Source.Repository,
						"revision": b.Source.Revision,
					},
					"subPath": b.Source.Path,
				},
				"tag": fmt.Sprintf("%s/%s/%s:latest", b.Configuration.DefaultImageRegistry.Host, function.Spec.HostRef.Name, function.Name),
				"additionalTags": []string{
					fmt.Sprintf("%s/%s/%s:%d", b.Configuration.DefaultImageRegistry.Host, function.Spec.HostRef.Name, function.Name, function.GetGeneration()),
				},
			},
		}

		return ctrl.SetControllerReference(function, kImage, b.Scheme)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create image builder: %w", err)
	}

	log := logf.FromContext(ctx)

	log.V(2).Info("GetOrCreateKPackImage", "op", op)

	return kImage, nil
}
