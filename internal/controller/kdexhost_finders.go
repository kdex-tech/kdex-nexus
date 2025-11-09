package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *KDexHostReconciler) findHostsForScriptLibrary(
	ctx context.Context,
	scriptLibrary client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var hostList kdexv1alpha1.KDexHostList
	if err := r.List(ctx, &hostList, &client.ListOptions{
		Namespace: scriptLibrary.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexHosts for scriptLibrary", "name", scriptLibrary.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(hostList.Items))
	for _, host := range hostList.Items {
		if host.Spec.ScriptLibraryRef == nil {
			continue
		}
		if host.Spec.ScriptLibraryRef.Name == scriptLibrary.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      host.Name,
					Namespace: host.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *KDexHostReconciler) findHostsForTheme(
	ctx context.Context,
	theme client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var hostList kdexv1alpha1.KDexHostList
	if err := r.List(ctx, &hostList, &client.ListOptions{
		Namespace: theme.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexHosts for theme", "name", theme.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(hostList.Items))
	for _, host := range hostList.Items {
		if host.Spec.DefaultThemeRef == nil {
			continue
		}
		if host.Spec.DefaultThemeRef.Name == theme.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      host.Name,
					Namespace: host.Namespace,
				},
			})
		}
	}
	return requests
}
