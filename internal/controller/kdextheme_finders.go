package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *KDexThemeReconciler) findThemesForScriptLibrary(
	ctx context.Context,
	scriptLibrary client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var themeList kdexv1alpha1.KDexThemeList
	if err := r.List(ctx, &themeList, &client.ListOptions{
		Namespace: scriptLibrary.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list KDexThemes for scriptLibrary", "scriptLibrary", scriptLibrary.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(themeList.Items))
	for _, theme := range themeList.Items {
		if theme.Spec.ScriptLibraryRef == nil {
			continue
		}
		if theme.Spec.ScriptLibraryRef.Name == scriptLibrary.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      theme.Name,
					Namespace: theme.Namespace,
				},
			})
		}
	}
	return requests
}
