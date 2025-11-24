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
	"kdex.dev/crds/base"
)

var _ = Describe("KDexTheme Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should not reconcile without assets", func() {
			resource := &kdexv1alpha1.KDexTheme{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexThemeSpec{},
			}

			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())
		})

		It("should reconcile with only absolute assets references", func() {
			resource := &kdexv1alpha1.KDexTheme{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{
							Attributes: map[string]string{
								"rel": "stylesheet",
							},
							LinkHref: "http://kdex.dev/style.css",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, true)
		})

		It("should not reconcile with relative assets but no image specified", func() {
			resource := &kdexv1alpha1.KDexTheme{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{
							Attributes: map[string]string{
								"rel": "stylesheet",
							},
							LinkHref: "/style.css",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, false)
		})

		It("should successfully reconcile after assets becomes valid", func() {
			addOrUpdateTheme(
				ctx, k8sClient,
				kdexv1alpha1.KDexTheme{
					KDexObject: base.KDexObject{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: namespace,
						},
					},
					Spec: kdexv1alpha1.KDexThemeSpec{
						Assets: []kdexv1alpha1.Asset{
							{
								Attributes: map[string]string{
									"!": `"`,
								},
								LinkHref: `"`,
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, false)

			addOrUpdateTheme(
				ctx, k8sClient,
				kdexv1alpha1.KDexTheme{
					KDexObject: base.KDexObject{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: namespace,
						},
					},
					Spec: kdexv1alpha1.KDexThemeSpec{
						Assets: []kdexv1alpha1.Asset{
							{
								Attributes: map[string]string{
									"rel": "stylesheet",
								},
								LinkHref: "http://kdex.dev/style.css",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, true)
		})

		It("should not reconcile with image but no routePath", func() {
			resource := &kdexv1alpha1.KDexTheme{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{
							Attributes: map[string]string{
								"rel": "stylesheet",
							},
							LinkHref: "/style.css",
						},
					},
					Image: "foo/bar",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, false)
		})

		It("should not reconcile with no image but routePath", func() {
			resource := &kdexv1alpha1.KDexTheme{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{
							Attributes: map[string]string{
								"rel": "stylesheet",
							},
							LinkHref: "/style.css",
						},
					},
					RoutePath: "/theme",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, false)
		})

		It("should not reconcile with image, routePath and relative assets that are not prefixed by routePath", func() {
			resource := &kdexv1alpha1.KDexTheme{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexThemeSpec{
					Assets: []kdexv1alpha1.Asset{
						{
							Attributes: map[string]string{
								"rel": "stylesheet",
							},
							LinkHref: "/style.css",
						},
					},
					Image:     "foo/bar",
					RoutePath: "/theme",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexTheme{}, false)
		})
	})
})
