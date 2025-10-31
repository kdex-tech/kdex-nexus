package controller

import (
	"context"
	"reflect"

	. "github.com/onsi/gomega"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addOrUpdatePageArchetype(
	ctx context.Context,
	k8sClient client.Client,
	pageArchetype kdexv1alpha1.KDexPageArchetype,
) {
	list := &kdexv1alpha1.KDexPageArchetypeList{}
	err := k8sClient.List(ctx, list, &client.ListOptions{
		Namespace:     pageArchetype.Namespace,
		FieldSelector: fields.OneTermEqualSelector("metadata.name", pageArchetype.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) > 0 {
		existing := list.Items[0]
		existing.Spec = pageArchetype.Spec
		Expect(k8sClient.Update(ctx, &existing)).To(Succeed())
	} else {
		Expect(k8sClient.Create(ctx, &pageArchetype)).To(Succeed())
	}
}

func addOrUpdatePageHeader(
	ctx context.Context,
	k8sClient client.Client,
	pageHeader kdexv1alpha1.KDexPageHeader,
) {
	list := &kdexv1alpha1.KDexPageHeaderList{}
	err := k8sClient.List(ctx, list, &client.ListOptions{
		Namespace:     pageHeader.Namespace,
		FieldSelector: fields.OneTermEqualSelector("metadata.name", pageHeader.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) > 0 {
		existing := list.Items[0]
		existing.Spec = pageHeader.Spec
		Expect(k8sClient.Update(ctx, &existing)).To(Succeed())
	} else {
		Expect(k8sClient.Create(ctx, &pageHeader)).To(Succeed())
	}
}

func addOrUpdatePageFooter(
	ctx context.Context,
	k8sClient client.Client,
	pageFooter kdexv1alpha1.KDexPageFooter,
) {
	list := &kdexv1alpha1.KDexPageFooterList{}
	err := k8sClient.List(ctx, list, &client.ListOptions{
		Namespace:     pageFooter.Namespace,
		FieldSelector: fields.OneTermEqualSelector("metadata.name", pageFooter.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) > 0 {
		existing := list.Items[0]
		existing.Spec = pageFooter.Spec
		Expect(k8sClient.Update(ctx, &existing)).To(Succeed())
	} else {
		Expect(k8sClient.Create(ctx, &pageFooter)).To(Succeed())
	}
}

func addOrUpdateHost(
	ctx context.Context,
	k8sClient client.Client,
	host kdexv1alpha1.KDexHost,
) {
	list := &kdexv1alpha1.KDexHostList{}
	err := k8sClient.List(ctx, list, &client.ListOptions{
		Namespace:     host.Namespace,
		FieldSelector: fields.OneTermEqualSelector("metadata.name", host.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) > 0 {
		existing := list.Items[0]
		existing.Spec = host.Spec
		Expect(k8sClient.Update(ctx, &existing)).To(Succeed())
	} else {
		Expect(k8sClient.Create(ctx, &host)).To(Succeed())
	}
}

func addOrUpdatePageNavigation(
	ctx context.Context,
	k8sClient client.Client,
	pageNavigation kdexv1alpha1.KDexPageNavigation,
) {
	list := &kdexv1alpha1.KDexPageNavigationList{}
	err := k8sClient.List(ctx, list, &client.ListOptions{
		Namespace:     pageNavigation.Namespace,
		FieldSelector: fields.OneTermEqualSelector("metadata.name", pageNavigation.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) > 0 {
		existing := list.Items[0]
		existing.Spec = pageNavigation.Spec
		Expect(k8sClient.Update(ctx, &existing)).To(Succeed())
	} else {
		Expect(k8sClient.Create(ctx, &pageNavigation)).To(Succeed())
	}
}

func addOrUpdateTheme(
	ctx context.Context,
	k8sClient client.Client,
	stylesheet kdexv1alpha1.KDexTheme,
) {
	list := &kdexv1alpha1.KDexThemeList{}
	err := k8sClient.List(ctx, list, &client.ListOptions{
		Namespace:     stylesheet.Namespace,
		FieldSelector: fields.OneTermEqualSelector("metadata.name", stylesheet.Name),
	})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) > 0 {
		existing := list.Items[0]
		existing.Spec = stylesheet.Spec
		Expect(k8sClient.Update(ctx, &existing)).To(Succeed())
	} else {
		Expect(k8sClient.Create(ctx, &stylesheet)).To(Succeed())
	}
}

func assertResourceReady(ctx context.Context, k8sClient client.Client, name string, namespace string, checkResource client.Object, ready bool) {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	check := func(g Gomega) {
		err := k8sClient.Get(ctx, typeNamespacedName, checkResource)
		g.Expect(err).NotTo(HaveOccurred())
		it := reflect.ValueOf(checkResource).Elem()
		statusField := it.FieldByName("Status")
		g.Expect(statusField.IsValid()).To(BeTrue())
		conditionsField := statusField.FieldByName("Conditions")
		g.Expect(conditionsField.IsValid()).To(BeTrue())
		conditions, ok := conditionsField.Interface().([]metav1.Condition)
		g.Expect(ok).To(BeTrue())
		if ready {
			g.Expect(
				apimeta.IsStatusConditionTrue(
					conditions, string(kdexv1alpha1.ConditionTypeReady),
				),
			).To(BeTrue())
		} else {
			g.Expect(
				apimeta.IsStatusConditionFalse(
					conditions, string(kdexv1alpha1.ConditionTypeReady),
				),
			).To(BeTrue())
		}
	}

	Eventually(check).Should(Succeed())
}
