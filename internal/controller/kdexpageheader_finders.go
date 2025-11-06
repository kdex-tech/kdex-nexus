package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *KDexPageHeaderReconciler) findPageHeadersForScriptLibrary(
	ctx context.Context,
	scriptLibrary client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageHeaderList kdexv1alpha1.KDexPageHeaderList
	if err := r.List(ctx, &pageHeaderList, &client.ListOptions{
		Namespace: scriptLibrary.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexPageHeader for scriptLibrary", "name", scriptLibrary.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(pageHeaderList.Items))
	for _, pageHeader := range pageHeaderList.Items {
		if pageHeader.Spec.ScriptLibraryRef == nil {
			continue
		}
		if pageHeader.Spec.ScriptLibraryRef.Name == scriptLibrary.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageHeader.Name,
					Namespace: pageHeader.Namespace,
				},
			})
		}
	}
	return requests
}
