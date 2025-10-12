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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/npm"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type MockRegistry struct{}

func (m *MockRegistry) ValidatePackage(packageName string, packageVersion string) error {
	if packageName == "@my-scope/missing" {
		return fmt.Errorf("package not found: %s", packageName)
	}

	return nil
}

var _ = Describe("MicroFrontEndApp Controller", Ordered, func() {
	BeforeAll(func() {
		By("Creating the reconciler")

		k8sManager, err := manager.New(cfg, manager.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		controllerReconciler := &MicroFrontEndAppReconciler{
			Client: k8sManager.GetClient(),
			RegistryFactory: func(
				secret *corev1.Secret,
				error func(err error, msg string, keysAndValues ...any),
			) npm.Registry {
				return &MockRegistry{}
			},
			RequeueDelay: 0,
			Scheme:       k8sManager.GetScheme(),
		}

		err = controllerReconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			err := k8sManager.Start(ctx)
			Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		}()
	})

	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		resourcesToDelete := map[types.NamespacedName]client.Object{}

		resourcesToDelete[types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}] = &kdexv1alpha1.MicroFrontEndApp{}

		AfterEach(func() {
			By("Cleanup all the test resource instances")
			for name, resource := range resourcesToDelete {
				err := k8sClient.Get(ctx, name, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("it must not become ready if it has missing package reference", func() {
			app := &kdexv1alpha1.MicroFrontEndApp{
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

			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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
			app := &kdexv1alpha1.MicroFrontEndApp{
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

			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			app := &kdexv1alpha1.MicroFrontEndApp{
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

			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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
			app := &kdexv1alpha1.MicroFrontEndApp{
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

			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			app := &kdexv1alpha1.MicroFrontEndApp{
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

			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			app := &kdexv1alpha1.MicroFrontEndApp{
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

			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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

			resourcesToDelete[types.NamespacedName{
				Name:      secret.Name,
				Namespace: namespace,
			}] = &corev1.Secret{}

			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			check = func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
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
