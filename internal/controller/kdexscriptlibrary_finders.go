package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *KDexScriptLibraryReconciler) findScriptLibrariesForSecret(
	ctx context.Context,
	secret client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	if _, ok := secret.GetAnnotations()["kdex.dev/npm-server-address"]; !ok {
		return []reconcile.Request{}
	}

	var scriptLibraryList kdexv1alpha1.KDexScriptLibraryList
	if err := r.List(ctx, &scriptLibraryList, &client.ListOptions{
		Namespace: secret.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexScriptLibraries for secret", "name", secret.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(scriptLibraryList.Items))
	for _, app := range scriptLibraryList.Items {
		if app.Spec.PackageReference == nil || app.Spec.PackageReference.SecretRef == nil {
			continue
		}
		if app.Spec.PackageReference.SecretRef.Name == secret.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
		}
	}
	return requests
}
