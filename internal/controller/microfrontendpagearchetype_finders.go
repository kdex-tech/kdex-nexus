package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *MicroFrontEndPageArchetypeReconciler) findPageArchetypesForPageFooter(
	ctx context.Context,
	pageFooter client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageArchetypesList kdexv1alpha1.MicroFrontEndPageArchetypeList
	if err := r.List(ctx, &pageArchetypesList, &client.ListOptions{
		Namespace: pageFooter.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageArchetypes for page footer %s", pageFooter.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(pageArchetypesList.Items))
	for _, pageArchetype := range pageArchetypesList.Items {
		if pageArchetype.Spec.DefaultFooterRef == nil {
			continue
		}
		if pageArchetype.Spec.DefaultFooterRef.Name == pageFooter.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageArchetype.Name,
					Namespace: pageArchetype.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *MicroFrontEndPageArchetypeReconciler) findPageArchetypesForPageHeader(
	ctx context.Context,
	pageHeader client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageArchetypesList kdexv1alpha1.MicroFrontEndPageArchetypeList
	if err := r.List(ctx, &pageArchetypesList, &client.ListOptions{
		Namespace: pageHeader.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageArchetypes for page header %s", pageHeader.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(pageArchetypesList.Items))
	for _, pageArchetype := range pageArchetypesList.Items {
		if pageArchetype.Spec.DefaultHeaderRef == nil {
			continue
		}
		if pageArchetype.Spec.DefaultHeaderRef.Name == pageHeader.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageArchetype.Name,
					Namespace: pageArchetype.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *MicroFrontEndPageArchetypeReconciler) findPageArchetypesForPageNavigations(
	ctx context.Context,
	pageNavigation client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageArchetypesList kdexv1alpha1.MicroFrontEndPageArchetypeList
	if err := r.List(ctx, &pageArchetypesList, &client.ListOptions{
		Namespace: pageNavigation.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageArchetypes for page navigation %s", pageNavigation.GetName())
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(pageArchetypesList.Items))
	for _, pageArchetype := range pageArchetypesList.Items {
		if pageArchetype.Spec.DefaultMainNavigationRef != nil {
			if pageArchetype.Spec.DefaultMainNavigationRef.Name == pageNavigation.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      pageArchetype.Name,
						Namespace: pageArchetype.Namespace,
					},
				})
			}
		}

		if pageArchetype.Spec.ExtraNavigations != nil {
			for _, navigationRef := range *pageArchetype.Spec.ExtraNavigations {
				if navigationRef.Name == pageNavigation.GetName() {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      pageArchetype.Name,
							Namespace: pageArchetype.Namespace,
						},
					})
				}
			}
		}
	}
	return requests
}
