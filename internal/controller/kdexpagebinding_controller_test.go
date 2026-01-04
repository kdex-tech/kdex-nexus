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
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexPageBinding Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
			cleanupResources(secondNamespace)
		})

		It("must not validate if basePath is empty", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.basePath in body should match '^/'`))
		})

		It("must not validate if no contentEntries are provided", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.contentEntries: Invalid value: "null"`))
		})

		It("must not validate if contentEntries is empty", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.contentEntries in body should have at least 1 items`))
		})

		It("must not validate if contentEntries doesn't have a 'main' slot", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{},
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`slot 'main' must be specified`))
		})

		It("must not validate if contentEntries doesn't have either 'rawHTML' or 'appRef'", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
						},
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`exactly one of the fields in [appRef rawHTML] must be set`))
		})

		It("must not validate if label is not set", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
						},
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.label in body should be at least 3 chars long`))
		})

		It("must not validate if contentEntries has both 'rawHTML' and 'appRef'", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryApp: kdexv1alpha1.ContentEntryApp{
								AppRef:            &kdexv1alpha1.KDexObjectReference{},
								CustomElementName: "test-custom-element",
							},
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "test-label",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "test-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`exactly one of the fields in [appRef rawHTML] must be set`))
		})

		It("must not validate if hostRef.name is empty", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<invalid html",
							},
						},
					},
					Label: "test",
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`hostRef.name must not be empty`))
		})

		It("must not validate if pageArchetypeRef.name is missing name", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<invalid html",
							},
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "test",
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`pageArchetypeRef.name must not be empty`))
		})

		It("must not validate if contentEntries has invalid rawHTML", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<invalid html",
							},
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "test",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "test-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`invalid go template in spec.contentEntries[0].rawHTML`))
		})

		It("must not validate if app contentEntry is missing customElementName", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryApp: kdexv1alpha1.ContentEntryApp{
								AppRef: &kdexv1alpha1.KDexObjectReference{},
							},
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "test",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "test-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`no such key: customElementName evaluating rule: appRef must be accompanied by customElementName`))
		})

		It("must not validate if app contentEntry appRef is missing name", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryApp: kdexv1alpha1.ContentEntryApp{
								AppRef:            &kdexv1alpha1.KDexObjectReference{},
								CustomElementName: "test-element",
							},
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "test",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "test-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`spec.contentEntries[0].appRef.name is required`))
		})

		It("will validate with minimum fields", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryApp: kdexv1alpha1.ContentEntryApp{
								AppRef: &kdexv1alpha1.KDexObjectReference{
									Name: "test-app",
								},
								CustomElementName: "test-element",
							},
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "test",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "test-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			err := k8sClient.Create(ctx, resource)
			Expect(err).ToNot(HaveOccurred())
		})

		It("with references should succeed", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			addOrUpdatePageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-page-archetype",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "non-existent-page-archetype", namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)

			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "test-host", namespace,
				&kdexv1alpha1.KDexHost{}, true)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, true)
		})

		It("with override references", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					OverrideFooterRef: &kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-footer",
					},
					OverrideHeaderRef: &kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-header",
					},
					OverrideNavigationRefs: map[string]*kdexv1alpha1.KDexObjectReference{
						"main": {
							Kind: "KDexPageNavigation",
							Name: "non-existent-navigation",
						},
					},
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/foo",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			addOrUpdatePageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-page-archetype",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)
			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)
			addOrUpdatePageFooter(ctx, k8sClient,
				kdexv1alpha1.KDexPageFooter{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-footer",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageFooterSpec{
						Content: "<h1>Hello, from down under!</h1>",
					},
				},
			)
			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-header",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "<h1>Hello, from up north!</h1>",
					},
				},
			)
			addOrUpdatePageNavigation(ctx, k8sClient,
				kdexv1alpha1.KDexPageNavigation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-navigation",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageNavigationSpec{
						Content: "<h1>Hello, from up north!</h1>",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, true)
		})

		It("with parent page reference", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-page-archetype",
					},
					ParentPageRef: &corev1.LocalObjectReference{
						Name: "non-existent-page-binding",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/child",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			addOrUpdatePageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-page-archetype",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)
			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			referencedPage := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-page-binding",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/",
					},
				},
			}

			Expect(k8sClient.Create(ctx, referencedPage)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, true)
		})

		It("updates when a dependency is updated", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					OverrideHeaderRef: &kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-header",
					},
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/foo",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			addOrUpdatePageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-page-archetype",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)
			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						ModulePolicy: kdexv1alpha1.LooseModulePolicy,
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)
			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-header",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "BEFORE",
					},
				},
			)

			checkedResource := &kdexv1alpha1.KDexPageBinding{}
			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				checkedResource, true)

			Expect(checkedResource.Status.Attributes["header.generation"]).To(Equal("1"))

			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-header",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "AFTER",
					},
				},
			)

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName,
					Namespace: namespace,
				}, checkedResource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(checkedResource.Status.Attributes["header.generation"]).To(Equal("2"))
			}

			Eventually(check, "5s").Should(Succeed())
		})

		It("updates when an indirect dependency is updated", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name: "non-existent-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/foo",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "test-brand",
						Organization: "test-organization",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"test-domain",
							},
						},
					},
				})

			assertResourceReady(
				ctx, k8sClient, "test-host", namespace,
				&kdexv1alpha1.KDexHost{}, true)

			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-header",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "BEFORE",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "non-existent-header", namespace,
				&kdexv1alpha1.KDexPageHeader{}, true)

			addOrUpdatePageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-page-archetype",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
						DefaultHeaderRef: &kdexv1alpha1.KDexObjectReference{
							Name: "non-existent-header",
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "non-existent-page-archetype", namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)

			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						ModulePolicy: kdexv1alpha1.LooseModulePolicy,
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "test-host", namespace,
				&kdexv1alpha1.KDexHost{}, true)

			checkedResource := &kdexv1alpha1.KDexPageBinding{}
			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				checkedResource, true)

			Expect(checkedResource.Status.Attributes["header.generation"]).To(Equal("1"))

			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-header",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "AFTER",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "non-existent-header", namespace,
				&kdexv1alpha1.KDexPageHeader{}, true)

			check := func(g Gomega) {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      resourceName,
					Namespace: namespace,
				}, checkedResource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(checkedResource.Status.Attributes["header.generation"]).To(Equal("2"))
			}

			Eventually(check, "5s").Should(Succeed())
		})

		It("cross namespace reference", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Name:      "non-existent-page-archetype",
						Namespace: secondNamespace,
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/foo",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "test-host", namespace,
				&kdexv1alpha1.KDexHost{}, true)

			addOrUpdatePageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-page-archetype",
						Namespace: secondNamespace,
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "non-existent-page-archetype", secondNamespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, true)
		})

		It("cluster reference", func() {
			resource := &kdexv1alpha1.KDexPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Hello, World!</h1>",
							},
							Slot: "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "test-host",
					},
					Label: "foo",
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Kind: "KDexClusterPageArchetype",
						Name: "non-existent-page-archetype",
					},
					Paths: kdexv1alpha1.Paths{
						BasePath: "/foo",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, false)

			addOrUpdateHost(
				ctx, k8sClient, kdexv1alpha1.KDexHost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-host",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexHostSpec{
						BrandName:    "KDex Tech",
						Organization: "KDex Tech Inc.",
						Routing: kdexv1alpha1.Routing{
							Domains: []string{
								"example.com",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "test-host", namespace,
				&kdexv1alpha1.KDexHost{}, true)

			addOrUpdateClusterPageArchetype(
				ctx, k8sClient,
				kdexv1alpha1.KDexClusterPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name: "non-existent-page-archetype",
					},
					Spec: kdexv1alpha1.KDexPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, "non-existent-page-archetype", "",
				&kdexv1alpha1.KDexClusterPageArchetype{}, true)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageBinding{}, true)
		})
	})
})
