package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *KDexPageFooterReconciler) findPageFootersForScriptLibrary(
	ctx context.Context,
	scriptLibrary client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageFooterList kdexv1alpha1.KDexPageFooterList
	if err := r.List(ctx, &pageFooterList, &client.ListOptions{
		Namespace: scriptLibrary.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexPageFooter for scriptLibrary", "name", scriptLibrary.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(pageFooterList.Items))
	for _, pageFooter := range pageFooterList.Items {
		if pageFooter.Spec.ScriptLibraryRef == nil {
			continue
		}
		if pageFooter.Spec.ScriptLibraryRef.Name == scriptLibrary.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageFooter.Name,
					Namespace: pageFooter.Namespace,
				},
			})
		}
	}
	return requests
}
