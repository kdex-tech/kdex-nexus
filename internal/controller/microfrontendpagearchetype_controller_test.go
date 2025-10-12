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
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("MicroFrontEndPageArchetype Controller", Ordered, func() {
	BeforeAll(func() {
		By("Creating the reconciler")

		k8sManager, err := manager.New(cfg, manager.Options{
			Metrics: server.Options{
				BindAddress: "0",
			},
			Scheme: scheme.Scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		controllerReconciler := &MicroFrontEndPageArchetypeReconciler{
			MicroFrontEndCommonReconciler: MicroFrontEndCommonReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			},
			RequeueDelay: 0,
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
		}] = &kdexv1alpha1.MicroFrontEndPageArchetype{}

		AfterEach(func() {
			By("Cleanup all the test resource instances")
			for name, resource := range resourcesToDelete {
				err := k8sClient.Get(ctx, name, resource)
				if errors.IsNotFound(err) {
					continue
				}
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("with missing extra navigation reference should not successfully reconcile the resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					ExtraNavigations: &map[string]corev1.LocalObjectReference{
						"non-existent-navigation": {
							Name: "non-existent-navigation",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionFalse(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())

			By("but when added should become ready")
			navigation := &kdexv1alpha1.MicroFrontEndPageNavigation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-navigation",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageNavigationSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, navigation)).To(Succeed())

			resourcesToDelete[types.NamespacedName{
				Name:      navigation.Name,
				Namespace: namespace,
			}] = &kdexv1alpha1.MicroFrontEndPageNavigation{}

			check = func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionTrue(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())
		})

		It("with missing default main navigation reference should not successfully reconcile the resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					DefaultMainNavigationRef: &corev1.LocalObjectReference{
						Name: "non-existent-main-navigation",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionFalse(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())

			By("but when added should become ready")
			navigation := &kdexv1alpha1.MicroFrontEndPageNavigation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-main-navigation",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageNavigationSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, navigation)).To(Succeed())

			resourcesToDelete[types.NamespacedName{
				Name:      navigation.Name,
				Namespace: namespace,
			}] = &kdexv1alpha1.MicroFrontEndPageNavigation{}

			check = func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionTrue(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())
		})

		It("with missing default footer reference should not successfully reconcile the resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					DefaultFooterRef: &corev1.LocalObjectReference{
						Name: "non-existent-footer",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionFalse(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())

			By("but when added should become ready")
			footer := &kdexv1alpha1.MicroFrontEndPageFooter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-footer",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageFooterSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, footer)).To(Succeed())

			resourcesToDelete[types.NamespacedName{
				Name:      footer.Name,
				Namespace: namespace,
			}] = &kdexv1alpha1.MicroFrontEndPageFooter{}

			check = func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionTrue(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())
		})

		It("with missing default header reference should not successfully reconcile the resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					DefaultHeaderRef: &corev1.LocalObjectReference{
						Name: "non-existent-header",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionFalse(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())

			By("but when added should become ready")
			header := &kdexv1alpha1.MicroFrontEndPageHeader{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-header",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageHeaderSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, header)).To(Succeed())

			resourcesToDelete[types.NamespacedName{
				Name:      header.Name,
				Namespace: namespace,
			}] = &kdexv1alpha1.MicroFrontEndPageFooter{}

			check = func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionTrue(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())
		})

		It("with only content should successfully reconcile the resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionTrue(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).Should(Succeed())
		})
	})
})
