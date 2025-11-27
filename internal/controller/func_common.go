package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/npm"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func validatePackageReference(
	ctx context.Context,
	packageReference *kdexv1alpha1.PackageReference,
	secret *corev1.Secret,
	registryFactory func(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) (npm.Registry, error),
) error {
	log := logf.FromContext(ctx)

	if !strings.HasPrefix(packageReference.Name, "@") || !strings.Contains(packageReference.Name, "/") {
		return fmt.Errorf("invalid package name, must be scoped with @scope/name: %s", packageReference.Name)
	}

	registry, err := registryFactory(secret, log.Error)
	if err != nil {
		return err
	}

	return registry.ValidatePackage(
		packageReference.Name,
		packageReference.Version,
	)
}
