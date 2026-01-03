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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexHost Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "host-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("it must not validate if it has missing brandName", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{},
			}

			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())
		})

		It("it must not validate if it has missing organization", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName: "KDex Tech",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())
		})

		It("it must not validate if it has missing routing", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())
		})

		It("it must not validate if it has missing routing domains", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())
		})

		It("it reconciles if minimum required fields are present", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"kdex.dev",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexHost{}, true)
		})

		It("it reconciles if theme reference becomes available", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"kdex.dev",
						},
						Strategy: kdexv1alpha1.IngressRoutingStrategy,
					},
					ThemeRef: &kdexv1alpha1.KDexObjectReference{
						Kind: "KDexTheme",
						Name: "non-existent-theme",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexHost{}, false)

			themeResource := &kdexv1alpha1.KDexTheme{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-theme",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: kdexv1alpha1.Assets{
						{
							LinkHref: "http://foo.bar/style.css",
							Attributes: map[string]string{
								"rel": "stylesheet",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, themeResource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, themeResource.Name, namespace,
				&kdexv1alpha1.KDexTheme{}, true)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexHost{}, true)
		})

		It("it reconciles if scriptlibrary reference becomes available", func() {
			resource := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					ModulePolicy: kdexv1alpha1.StrictModulePolicy,
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"kdex.dev",
						},
						Strategy: kdexv1alpha1.IngressRoutingStrategy,
					},
					ScriptLibraryRef: &kdexv1alpha1.KDexObjectReference{
						Kind: "KDexScriptLibrary",
						Name: "non-existent-script-library",
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexHost{}, false)

			addOrUpdateScriptLibrary(
				ctx, k8sClient,
				kdexv1alpha1.KDexScriptLibrary{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "non-existent-script-library",
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexScriptLibrarySpec{
						Scripts: []kdexv1alpha1.ScriptDef{
							{
								Attributes: map[string]string{
									"type": "text/module",
								},
								ScriptSrc: "http://foo.bar/script.js",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexHost{}, true)
		})

		It("it reconciles a referenced utility page", func() {
			host := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"kdex.dev",
						},
					},
					UtilityPages: &kdexv1alpha1.UtilityPages{
						AnnouncementRef: &kdexv1alpha1.KDexObjectReference{
							Kind: "KDexUtilityPage",
							Name: "non-existent-announcement",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, host)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, host.Name, namespace,
				&kdexv1alpha1.KDexHost{}, false)

			pageArchetype := &kdexv1alpha1.KDexPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-page-archetype",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "{{ .Content.main }}",
				},
			}

			Expect(k8sClient.Create(ctx, pageArchetype)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, pageArchetype.Name, namespace,
				&kdexv1alpha1.KDexPageArchetype{}, true)

			announcementPage := &kdexv1alpha1.KDexUtilityPage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-announcement",
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexUtilityPageSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Announcement</h1>",
							},
						},
					},
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Kind: "KDexPageArchetype",
						Name: "non-existent-page-archetype",
					},
					Type: kdexv1alpha1.AnnouncementUtilityPageType,
				},
			}

			Expect(k8sClient.Create(ctx, announcementPage)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, announcementPage.Name, namespace,
				&kdexv1alpha1.KDexUtilityPage{}, true)

			checkedHost := &kdexv1alpha1.KDexHost{}
			assertResourceReady(
				ctx, k8sClient, host.Name, namespace,
				checkedHost, true)

			Eventually(
				checkedHost.Status.Attributes["announcement.utilitypage.generation"], "5s",
			).Should(Equal("1"))

			internalUtilityPage := &kdexv1alpha1.KDexInternalUtilityPage{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-announcement", host.Name),
				Namespace: namespace,
			}, internalUtilityPage)
			Expect(err).NotTo(HaveOccurred())
			Expect(internalUtilityPage.Spec.ContentEntries).To(HaveLen(1))
			Expect(internalUtilityPage.Spec.PageArchetypeRef.Name).To(Equal(pageArchetype.Name))
			Expect(internalUtilityPage.Spec.Type).To(Equal(kdexv1alpha1.AnnouncementUtilityPageType))
			Expect(internalUtilityPage.Spec.ContentEntries[0].Slot).To(Equal("main"))
			Expect(internalUtilityPage.Spec.ContentEntries[0].ContentEntryStatic.RawHTML).To(Equal("<h1>Announcement</h1>"))
		})

		It("it reconciles a default utility page if a default is available and none is specified", func() {
			pageArchetype := &kdexv1alpha1.KDexClusterPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kdex-default-page-archetype",
				},
				Spec: kdexv1alpha1.KDexPageArchetypeSpec{
					Content: "{{ .Content.main }}",
				},
			}

			Expect(k8sClient.Create(ctx, pageArchetype)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, pageArchetype.Name, "",
				&kdexv1alpha1.KDexClusterPageArchetype{}, true)

			announcementPage := &kdexv1alpha1.KDexClusterUtilityPage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kdex-default-utility-page-announcement",
				},
				Spec: kdexv1alpha1.KDexUtilityPageSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							Slot: "main",
							ContentEntryStatic: kdexv1alpha1.ContentEntryStatic{
								RawHTML: "<h1>Announcement</h1>",
							},
						},
					},
					PageArchetypeRef: kdexv1alpha1.KDexObjectReference{
						Kind: "KDexClusterPageArchetype",
						Name: pageArchetype.Name,
					},
					Type: kdexv1alpha1.AnnouncementUtilityPageType,
				},
			}

			Expect(k8sClient.Create(ctx, announcementPage)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, announcementPage.Name, "",
				&kdexv1alpha1.KDexClusterUtilityPage{}, true)

			host := &kdexv1alpha1.KDexHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexHostSpec{
					BrandName:    "KDex Tech",
					Organization: "KDex Tech Inc.",
					Routing: kdexv1alpha1.Routing{
						Domains: []string{
							"kdex.dev",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, host)).To(Succeed())

			checkedHost := &kdexv1alpha1.KDexHost{}
			assertResourceReady(
				ctx, k8sClient, host.Name, namespace,
				checkedHost, true)

			Eventually(
				checkedHost.Status.Attributes["announcement.utilitypage.generation"], "5s",
			).Should(Equal("1"))

			internalUtilityPage := &kdexv1alpha1.KDexInternalUtilityPage{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-announcement", host.Name),
				Namespace: namespace,
			}, internalUtilityPage)
			Expect(err).NotTo(HaveOccurred())
			Expect(internalUtilityPage.Spec.ContentEntries).To(HaveLen(1))
			Expect(internalUtilityPage.Spec.PageArchetypeRef.Name).To(Equal(pageArchetype.Name))
			Expect(internalUtilityPage.Spec.Type).To(Equal(kdexv1alpha1.AnnouncementUtilityPageType))
			Expect(internalUtilityPage.Spec.ContentEntries[0].Slot).To(Equal("main"))
			Expect(internalUtilityPage.Spec.ContentEntries[0].ContentEntryStatic.RawHTML).To(Equal("<h1>Announcement</h1>"))
		})
	})
})
