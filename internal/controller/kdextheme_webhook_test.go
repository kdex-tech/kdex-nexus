package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexTheme Webhook", func() {
	Context("When creating a KDexTheme", func() {
		const namespace = "default"
		const resourceName = "test-theme-webhook-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should default IngressPath if missing", func() {
			resource := &kdexv1alpha1.KDexTheme{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{LinkHref: "styles.css"},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexTheme{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.WebServer.IngressPath).To(Equal("/theme"))
		})

		It("should not overwrite IngressPath if present", func() {
			resource := &kdexv1alpha1.KDexTheme{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{LinkHref: "styles.css"},
					},
					WebServer: kdexv1alpha1.WebServer{
						IngressPath: "/custom",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexTheme{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.WebServer.IngressPath).To(Equal("/custom"))
		})
	})
})
