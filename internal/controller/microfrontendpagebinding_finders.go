package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *MicroFrontEndPageBindingReconciler) findPageBindingsForApp(
	ctx context.Context,
	app client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageBindingsList kdexv1alpha1.MicroFrontEndPageBindingList
	if err := r.List(ctx, &pageBindingsList, &client.ListOptions{
		Namespace: app.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageBindings for app", "name", app.GetName())
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, pageBinding := range pageBindingsList.Items {
		for _, contentEntry := range pageBinding.Spec.ContentEntries {
			if contentEntry.AppRef == nil {
				continue
			}
			if contentEntry.AppRef.Name == app.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      pageBinding.Name,
						Namespace: pageBinding.Namespace,
					},
				})
			}
		}
	}
	return requests
}

func (r *MicroFrontEndPageBindingReconciler) findPageBindingsForHost(
	ctx context.Context,
	host client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageBindingsList kdexv1alpha1.MicroFrontEndPageBindingList
	if err := r.List(ctx, &pageBindingsList, &client.ListOptions{
		Namespace: host.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageBindings for host", "name", host.GetName())
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, pageBinding := range pageBindingsList.Items {
		if pageBinding.Spec.HostRef.Name == host.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *MicroFrontEndPageBindingReconciler) findPageBindingsForPageArchetype(
	ctx context.Context,
	pageArchetype client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageBindingsList kdexv1alpha1.MicroFrontEndPageBindingList
	if err := r.List(ctx, &pageBindingsList, &client.ListOptions{
		Namespace: pageArchetype.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageBindings for page archetype", "name", pageArchetype.GetName())
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, pageBinding := range pageBindingsList.Items {
		if pageBinding.Spec.PageArchetypeRef.Name == pageArchetype.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *MicroFrontEndPageBindingReconciler) findPageBindingsForPageFooter(
	ctx context.Context,
	pageFooter client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageBindingsList kdexv1alpha1.MicroFrontEndPageBindingList
	if err := r.List(ctx, &pageBindingsList, &client.ListOptions{
		Namespace: pageFooter.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageBindings for page footer", "name", pageFooter.GetName())
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, pageBinding := range pageBindingsList.Items {
		if pageBinding.Spec.OverrideFooterRef == nil {
			continue
		}
		if pageBinding.Spec.OverrideFooterRef.Name == pageFooter.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *MicroFrontEndPageBindingReconciler) findPageBindingsForPageHeader(
	ctx context.Context,
	pageHeader client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageBindingsList kdexv1alpha1.MicroFrontEndPageBindingList
	if err := r.List(ctx, &pageBindingsList, &client.ListOptions{
		Namespace: pageHeader.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageBindings for page header", "name", pageHeader.GetName())
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, pageBinding := range pageBindingsList.Items {
		if pageBinding.Spec.OverrideHeaderRef == nil {
			continue
		}
		if pageBinding.Spec.OverrideHeaderRef.Name == pageHeader.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *MicroFrontEndPageBindingReconciler) findPageBindingsForPageNavigations(
	ctx context.Context,
	pageNavigation client.Object,
) []reconcile.Request {
	log := logf.FromContext(ctx)

	var pageBindingsList kdexv1alpha1.MicroFrontEndPageBindingList
	if err := r.List(ctx, &pageBindingsList, &client.ListOptions{
		Namespace: pageNavigation.GetNamespace(),
	}); err != nil {
		log.Error(err, "unable to list MicroFrontEndPageBindings for page navigation", "name", pageNavigation.GetName())
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, pageBinding := range pageBindingsList.Items {
		if pageBinding.Spec.OverrideMainNavigationRef == nil {
			continue
		}
		if pageBinding.Spec.OverrideMainNavigationRef.Name == pageNavigation.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pageBinding.Name,
					Namespace: pageBinding.Namespace,
				},
			})
		}
	}
	return requests
}
