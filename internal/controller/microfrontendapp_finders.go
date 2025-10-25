package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *MicroFrontEndAppReconciler) findAppsForSecret(
	ctx context.Context,
	secret client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	if _, ok := secret.GetAnnotations()["kdex.dev/npm-server-address"]; !ok {
		return []reconcile.Request{}
	}

	var appList kdexv1alpha1.MicroFrontEndAppList
	if err := r.List(ctx, &appList, &client.ListOptions{
		Namespace: secret.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndApps for secret", "name", secret.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(appList.Items))
	for _, app := range appList.Items {
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
