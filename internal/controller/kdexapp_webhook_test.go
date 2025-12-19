package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexApp Webhook", func() {
	Context("When creating a KDexApp", func() {
		const namespace = "default"
		const resourceName = "test-webhook-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should default IngressPath if missing", func() {
			resource := &kdexv1alpha1.KDexApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexAppSpec{
					// Minimal required fields
					CustomElements: []kdexv1alpha1.CustomElement{
						{Name: "foo", Description: "bar"},
					},
				},
			}

			// IngressPath is empty by default
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexApp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.WebServer.IngressPath).To(Equal("/" + resourceName))
		})

		It("should not overwrite IngressPath if present", func() {
			resource := &kdexv1alpha1.KDexApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{Name: "foo", Description: "bar"},
					},
					WebServer: kdexv1alpha1.WebServer{
						IngressPath: "/custom",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexApp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.WebServer.IngressPath).To(Equal("/custom"))
		})
	})
})
