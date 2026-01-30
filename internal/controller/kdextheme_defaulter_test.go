package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexTheme Defaulter", func() {
	Context("When creating a KDexTheme", func() {
		const resourceName = "test-theme-webhook-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should default IngressPath", func() {
			resource := &kdexv1alpha1.KDexTheme{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{LinkHref: "http://foo.bar/styles.css"},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexTheme{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.IngressPath).To(Equal("/-/theme"))
		})

		It("should overwrite IngressPath if set", func() {
			resource := &kdexv1alpha1.KDexTheme{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{LinkHref: "/-/theme/styles.css"},
					},
					Backend: kdexv1alpha1.Backend{
						IngressPath: "/custom",
						StaticImage: "kdex/theme:1.2.3",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexTheme{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.IngressPath).To(Equal("/-/theme"))
		})
	})
})
