/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MicroFrontEndApp Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		AfterEach(func() {
			By("Cleanup all the test resource instances")
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndApp{}, client.InNamespace(namespace))).To(Succeed())

			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndHost{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndPageArchetype{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndPageBinding{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndPageFooter{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndPageHeader{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndPageNavigation{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndStylesheet{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.MicroFrontEndTranslation{}, client.InNamespace(namespace))).To(Succeed())
		})

		It("it must not become ready if it has missing package reference", func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{
							Description: "",
							Name:        "foo",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionFalse),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal("PackageValidationFailed"),
				)
				g.Expect(
					condition.Message,
				).To(
					ContainSubstring("invalid package name, must be scoped with @scope/name:"),
				)
			}

			Eventually(check).Should(Succeed())
		})

		It("it should become ready if it has a valid package reference", func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{
							Description: "",
							Name:        "foo",
						},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "@my-scope/my-package",
						Version: "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionTrue),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal(string(kdexv1alpha1.ConditionReasonReconcileSuccess)),
				)
				g.Expect(
					condition.Message,
				).To(
					Equal("all references resolved successfully"),
				)
			}

			Eventually(check).Should(Succeed())
		})

		It("should not become ready if it has a unscoped package reference", func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{
							Description: "",
							Name:        "foo",
						},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "my-scope/my-package",
						Version: "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionFalse),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal("PackageValidationFailed"),
				)
				g.Expect(
					condition.Message,
				).To(
					ContainSubstring("invalid package name, must be scoped with @scope/name:"),
				)
			}

			Eventually(check).Should(Succeed())
		})

		It("it must not become ready if it has a valid package reference but the package is missing", func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{
							Description: "",
							Name:        "foo",
						},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name:    "@my-scope/missing",
						Version: "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionFalse),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal("PackageValidationFailed"),
				)
				g.Expect(
					condition.Message,
				).To(
					Equal("package not found: @my-scope/missing"),
				)
			}

			Eventually(check).Should(Succeed())
		})

		It("should not become ready when referenced secret is not found", func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{
							Description: "",
							Name:        "foo",
						},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name: "my-scope/my-package",
						SecretRef: &corev1.LocalObjectReference{
							Name: "non-existent-secret",
						},
						Version: "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionFalse),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal("ReconcileError"),
				)
				g.Expect(
					condition.Message,
				).To(
					Equal("referenced Secret non-existent-secret not found"),
				)
			}

			Eventually(check).Should(Succeed())
		})

		It("should become ready when referenced secret is found", func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndAppSpec{
					CustomElements: []kdexv1alpha1.CustomElement{
						{
							Description: "",
							Name:        "foo",
						},
					},
					PackageReference: kdexv1alpha1.PackageReference{
						Name: "@my-scope/my-package",
						SecretRef: &corev1.LocalObjectReference{
							Name: "existent-secret",
						},
						Version: "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionFalse),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal("ReconcileError"),
				)
				g.Expect(
					condition.Message,
				).To(
					Equal("referenced Secret existent-secret not found"),
				)
			}

			Eventually(check).Should(Succeed())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kdex.dev/npm-server-address": "https://registry.npmjs.org",
					},
					Name:      "existent-secret",
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"username": []byte("username"),
					"password": []byte("password"),
				},
			}

			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			check = func(g Gomega) {
				checkResource := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, checkResource)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					checkResource.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionTrue),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal(string(kdexv1alpha1.ConditionReasonReconcileSuccess)),
				)
				g.Expect(
					condition.Message,
				).To(
					Equal("all references resolved successfully"),
				)
			}

			Eventually(check).Should(Succeed())
		})
	})
})
