package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexApp Validator", func() {
	Context("When creating a KDexApp", func() {
		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should fail with invalid package name", func() {
			resource := &kdexv1alpha1.KDexApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-validator-invalid-package",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexAppSpec{
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "invalid-package", // not scoped
						Version: "1.0.0",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.packageReference.name in body should match"))
		})

		It("should fail with relative URL and no image", func() {
			resource := &kdexv1alpha1.KDexApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-validator-relative-url",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{Name: "dummy-element", Description: "required by validation"},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "@valid/package",
						Version: "1.0.0",
					},
					Scripts: []kdexv1alpha1.ScriptDef{
						{
							ScriptSrc: "/some/relative/path",
						},
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("contains relative url but no image was provided"))
		})
	})
})
