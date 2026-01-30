package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexHost Defaulter", func() {
	Context("When creating a KDexHost", func() {
		const resourceName = "test-host-webhook-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should default fields if missing", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{"kdex.dev"},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexHost{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.DefaultLang).To(Equal("en"))
			Expect(createdResource.Spec.ModulePolicy).To(Equal(kdexv1alpha1.StrictModulePolicy))
			Expect(createdResource.Spec.IngressPath).To(Equal("/-/host"))
		})

		It("should not overwrite fields if present except ingressPath", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{"kdex.dev"},
					},
					DefaultLang:  "fr",
					ModulePolicy: kdexv1alpha1.StrictModulePolicy, // Can verify it doesn't change
					Backend: kdexv1alpha1.Backend{
						IngressPath: "/custom",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createdResource := &kdexv1alpha1.KDexHost{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, createdResource)).To(Succeed())

			Expect(createdResource.Spec.DefaultLang).To(Equal("fr"))
			Expect(createdResource.Spec.IngressPath).To(Equal("/-/host"))
		})
	})
})
