package validation

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/npm"
	"kdex.dev/crds/render"
	kdexresource "kdex.dev/crds/resource"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func ValidatePackageReference(
	ctx context.Context,
	packageReference *kdexv1alpha1.PackageReference,
	secret *corev1.Secret,
	registryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) (npm.Registry, error),
) error {
	log := logf.FromContext(ctx)

	registry, err := registryFactory(secret, log.Error)
	if err != nil {
		return err
	}

	return registry.ValidatePackage(
		packageReference.Name,
		packageReference.Version,
	)
}

func ValidateAssets(assets kdexv1alpha1.Assets) error {
	renderer := render.Renderer{}

	_, err := renderer.RenderOne(
		"theme-assets",
		assets.String(),
		render.DefaultTemplateData(),
	)

	return err
}

func ValidateResourceProvider(resourceProvider kdexresource.ResourceProvider) error {
	if resourceProvider.GetResourceImage() == "" {
		for _, url := range resourceProvider.GetResourceURLs() {
			if url != "" && !strings.Contains(url, "://") {
				return fmt.Errorf("%s contains relative url but no image was provided", url)
			}
		}
	}

	if resourceProvider.GetResourceImage() != "" && resourceProvider.GetResourcePath() == "" {
		return fmt.Errorf("ingressPath must be specified when an image is specified")
	}

	if resourceProvider.GetResourceImage() != "" && resourceProvider.GetResourcePath() != "" {
		for _, url := range resourceProvider.GetResourceURLs() {
			if url != "" &&
				!strings.Contains(url, "://") &&
				!strings.HasPrefix(url, resourceProvider.GetResourcePath()) {

				return fmt.Errorf("%s is not prefixed by ingressPath: %s", url, resourceProvider.GetResourcePath())
			}
		}
	}

	return nil
}
