package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexHost Validator", func() {
	Context("When creating a KDexHost", func() {
		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should fail without domains", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.routing.domains in body must be of type array"))
		})

		It("should fail without organization", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"invalid",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.organization: Invalid value: "": spec.organization in body should be at least 5 chars long`))
		})

		It("should fail with invalid organization", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Organization: "123",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"invalid",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.organization: Invalid value: "123": spec.organization in body should be at least 5 chars long`))
		})

		It("should fail without brandName", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Organization: "valid",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"invalid",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.brandName: Invalid value: ""`))
		})

		It("should fail with invalid assets", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Assets: kdexv1alpha1.Assets{
						{
							LinkHref: "http://{{",
						},
					},
					BrandName:    "valid",
					Organization: "valid",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"invalid",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`template: theme-assets:1: unterminated quoted string`))
		})

		It("should fail with relative assets but no static image", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Assets: kdexv1alpha1.Assets{
						{
							LinkHref: "/foo",
						},
					},
					BrandName:    "valid",
					Organization: "valid",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"valid",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`/foo contains relative url but no image was provided`))
		})

		It("should fail with relative assets, static image, but wrong prefix", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Assets: kdexv1alpha1.Assets{
						{
							LinkHref: "/foo",
						},
					},
					BrandName:    "valid",
					Organization: "valid",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"valid",
						},
					},
					Backend: kdexv1alpha1.Backend{
						StaticImage: "kdex/static",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`/foo is not prefixed by ingressPath: /_host`))
		})

		It("should succeed with relative assets, static image and correct prefix", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					Assets: kdexv1alpha1.Assets{
						{
							LinkHref: "/_host/foo.css",
						},
					},
					BrandName:    "valid",
					Organization: "valid",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"valid",
						},
					},
					Backend: kdexv1alpha1.Backend{
						StaticImage: "kdex/static",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
