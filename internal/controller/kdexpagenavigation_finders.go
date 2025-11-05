package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *KDexPageNavigationReconciler) findPageNavigationsForScriptLibrary(
	ctx context.Context,
	scriptLibrary client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageNavigationList kdexv1alpha1.KDexPageNavigationList
	if err := r.List(ctx, &pageNavigationList, &client.ListOptions{
		Namespace: scriptLibrary.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexPageNavigations for scriptLibrary", "name", scriptLibrary.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(pageNavigationList.Items))
	for _, pageNavigation := range pageNavigationList.Items {
		if pageNavigation.Spec.ScriptLibraryRef == nil {
			continue
		}
		if pageNavigation.Spec.ScriptLibraryRef.Name == scriptLibrary.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageNavigation.Name,
					Namespace: pageNavigation.Namespace,
				},
			})
		}
	}
	return requests
}
