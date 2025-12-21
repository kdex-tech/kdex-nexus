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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

var _ = Describe("KDexHost Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
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
	})
})
