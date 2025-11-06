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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("KDexPageArchetype Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		AfterEach(func() {
			By("Cleanup all the test resource instances")
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.KDexPageArchetype{}, client.InNamespace(namespace))).To(Succeed())

			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.KDexPageFooter{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.KDexPageHeader{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.KDexPageNavigation{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &kdexv1alpha1.KDexTheme{}, client.InNamespace(namespace))).To(Succeed())
		})

		It("with invalid content will not reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>{{ !?$ }}</h1>",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)
		})

		It("with missing extra navigation reference should not successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					ExtraNavigations: map[string]*corev1.LocalObjectReference{
						"non-existent-navigation": {
							Name: "non-existent-navigation",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)

			By("but when added should become ready")
			navigation := &kdexv1alpha1.KDexPageNavigation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-navigation",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageNavigationSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, navigation)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})

		It("with missing default main navigation reference should not successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					DefaultMainNavigationRef: &corev1.LocalObjectReference{
						Name: "non-existent-main-navigation",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)

			By("but when added should become ready")
			navigation := &kdexv1alpha1.KDexPageNavigation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-main-navigation",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageNavigationSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, navigation)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})

		It("with missing default footer reference should not successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					DefaultFooterRef: &corev1.LocalObjectReference{
						Name: "non-existent-footer",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)

			By("but when added should become ready")
			footer := &kdexv1alpha1.KDexPageFooter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-footer",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageFooterSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, footer)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})

		It("with missing default header reference should not successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					DefaultHeaderRef: &corev1.LocalObjectReference{
						Name: "non-existent-header",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)

			By("but when added should become ready")
			header := &kdexv1alpha1.KDexPageHeader{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-header",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageHeaderSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}
			Expect(k8sClient.Create(ctx, header)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})

		It("with missing theme reference should not successfully reconcile", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					OverrideThemeRef: &corev1.LocalObjectReference{
						Name: "non-existent-theme",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)

			addOrUpdateTheme(
				ctx, k8sClient,
				kdexv1alpha1.KDexTheme{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-theme",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexThemeSpec{
						Assets: []kdexv1alpha1.Asset{
							{
								LinkHref: "http://foo.bar/style.css",
								Attributes: map[string]string{
									"rel": "stylesheet",
								},
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})

		It("with only content should successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})

		It("should successfully reconcile after script library becomes available", func() {
			resource := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "<h1>Hello, World!</h1>",
					ScriptLibraryRef: &corev1.LocalObjectReference{
						Name: "none-existent-script-library",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, false)

			addOrUpdateScriptLibrary(
				ctx, k8sClient,
				kdexv1alpha1.KDexScriptLibrary{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "none-existent-script-library",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexScriptLibrarySpec{
						Scripts: []kdexv1alpha1.Script{
							{
								Script: "console.log('test');",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)
		})
	})
})
