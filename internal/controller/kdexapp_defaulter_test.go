package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexApp Defaulter", func() {
	Context("When creating a KDexApp", func() {
		const resourceName = "test-webhook-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should default IngressPath", func() {
			resource := &kdexv1alpha1.KDexApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{Name: "foo", Description: "bar"},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "@foo/bar",
						Version: "1.2.3",
					},
				},
			}

			// IngressPath is empty by default
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexApp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.IngressPath).To(Equal("/_a/" + resourceName))
		})

		It("should overwrite IngressPath if set", func() {
			resource := &kdexv1alpha1.KDexApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexAppSpec{
					Backend: kdexv1alpha1.Backend{
						IngressPath: "/custom",
					},
					CustomElements: []kdexv1alpha1.CustomElement{
						{Name: "foo", Description: "bar"},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "@foo/bar",
						Version: "1.2.3",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexApp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.IngressPath).To(Equal("/_a/" + resourceName))
		})
	})
})
