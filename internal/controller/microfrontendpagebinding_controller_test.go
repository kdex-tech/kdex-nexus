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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MicroFrontEndPageBinding Controller", Ordered, func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		resourcesToDelete := map[types.NamespacedName]client.Object{}

		AfterEach(func() {
			By("Cleanup all the test resource instances")
			for name, resource := range resourcesToDelete {
				err := k8sClient.Get(ctx, name, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
				delete(resourcesToDelete, name)
			}
		})

		It("with empty content entries should not succeed", func() {
			resource := &kdexv1alpha1.MicroFrontEndPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{},
					HostRef: corev1.LocalObjectReference{
						Name: "non-existent-host",
					},
					Label: "foo",
					PageArchetypeRef: corev1.LocalObjectReference{
						Name: "non-existent-page-archetype",
					},
					Path: "/foo",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())
		})

		It("with content entries should succeed", func() {
			resource := &kdexv1alpha1.MicroFrontEndPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							RawHTML: "<h1>Hello, World!</h1>",
							Slot:    "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "non-existent-host",
					},
					Label: "foo",
					PageArchetypeRef: corev1.LocalObjectReference{
						Name: "non-existent-page-archetype",
					},
					Path: "/foo",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			resourcesToDelete[types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}] = &kdexv1alpha1.MicroFrontEndPageBinding{}
		})

		It("with missing references should not succeed", func() {
			resource := &kdexv1alpha1.MicroFrontEndPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							RawHTML: "<h1>Hello, World!</h1>",
							Slot:    "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "non-existent-host",
					},
					Label: "foo",
					PageArchetypeRef: corev1.LocalObjectReference{
						Name: "non-existent-page-archetype",
					},
					Path: "/foo",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			resourcesToDelete[typeNamespacedName] = &kdexv1alpha1.MicroFrontEndPageBinding{}

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.MicroFrontEndPageBinding{}, false)

			By("and when Host added should still not become ready")
			addHost(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-host")
			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.MicroFrontEndPageBinding{}, false)

			By("lastly when PageArchetype added should become ready")
			addPageArchetype(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-page-archetype")
			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.MicroFrontEndPageBinding{}, true)
		})

		It("with override footer reference", func() {
			resource := &kdexv1alpha1.MicroFrontEndPageBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.MicroFrontEndPageBindingSpec{
					ContentEntries: []kdexv1alpha1.ContentEntry{
						{
							RawHTML: "<h1>Hello, World!</h1>",
							Slot:    "main",
						},
					},
					HostRef: corev1.LocalObjectReference{
						Name: "non-existent-host",
					},
					Label: "foo",
					OverrideFooterRef: &corev1.LocalObjectReference{
						Name: "non-existent-footer",
					},
					OverrideHeaderRef: &corev1.LocalObjectReference{
						Name: "non-existent-header",
					},
					OverrideMainNavigationRef: &corev1.LocalObjectReference{
						Name: "non-existent-navigation",
					},
					PageArchetypeRef: corev1.LocalObjectReference{
						Name: "non-existent-page-archetype",
					},
					Path: "/foo",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			resourcesToDelete[typeNamespacedName] = &kdexv1alpha1.MicroFrontEndPageBinding{}

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.MicroFrontEndPageBinding{}, false)

			By("adding all the missing references")
			addHost(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-host")
			addPageArchetype(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-page-archetype")
			addPageFooter(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-footer")
			addPageHeader(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-header")
			addPageNavigation(ctx, k8sClient, resourcesToDelete, namespace, "non-existent-navigation")

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.MicroFrontEndPageBinding{}, true)
		})
	})
})

func addHost(
	ctx context.Context,
	k8sClient client.Client,
	resourcesToDelete map[types.NamespacedName]client.Object,
	namespace string,
	name string,
) {
	host := &kdexv1alpha1.MicroFrontEndHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kdexv1alpha1.MicroFrontEndHostSpec{
			AppPolicy: kdexv1alpha1.NonStrictAppPolicy,
			Domains: []string{
				"example.com",
			},
			Organization: "KDex Tech",
			Stylesheet:   "http://example.com/style.css",
		},
	}
	Expect(k8sClient.Create(ctx, host)).To(Succeed())

	resourcesToDelete[types.NamespacedName{
		Name:      host.Name,
		Namespace: namespace,
	}] = &kdexv1alpha1.MicroFrontEndHost{}
}

func addPageArchetype(
	ctx context.Context,
	k8sClient client.Client,
	resourcesToDelete map[types.NamespacedName]client.Object,
	namespace string,
	name string,
) {
	pageArchetype := &kdexv1alpha1.MicroFrontEndPageArchetype{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
			Content: "<h1>Hello, World!</h1>",
		},
	}
	Expect(k8sClient.Create(ctx, pageArchetype)).To(Succeed())

	resourcesToDelete[types.NamespacedName{
		Name:      pageArchetype.Name,
		Namespace: namespace,
	}] = &kdexv1alpha1.MicroFrontEndPageArchetype{}
}

func addPageFooter(
	ctx context.Context,
	k8sClient client.Client,
	resourcesToDelete map[types.NamespacedName]client.Object,
	namespace string,
	name string,
) {
	pageFooter := &kdexv1alpha1.MicroFrontEndPageFooter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kdexv1alpha1.MicroFrontEndPageFooterSpec{
			Content: "<h1>Hello, from down under!</h1>",
		},
	}
	Expect(k8sClient.Create(ctx, pageFooter)).To(Succeed())

	resourcesToDelete[types.NamespacedName{
		Name:      pageFooter.Name,
		Namespace: namespace,
	}] = &kdexv1alpha1.MicroFrontEndPageFooter{}
}

func addPageHeader(
	ctx context.Context,
	k8sClient client.Client,
	resourcesToDelete map[types.NamespacedName]client.Object,
	namespace string,
	name string,
) {
	pageHeader := &kdexv1alpha1.MicroFrontEndPageHeader{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kdexv1alpha1.MicroFrontEndPageHeaderSpec{
			Content: "<h1>Hello, from up north!</h1>",
		},
	}
	Expect(k8sClient.Create(ctx, pageHeader)).To(Succeed())

	resourcesToDelete[types.NamespacedName{
		Name:      pageHeader.Name,
		Namespace: namespace,
	}] = &kdexv1alpha1.MicroFrontEndPageHeader{}
}

func addPageNavigation(
	ctx context.Context,
	k8sClient client.Client,
	resourcesToDelete map[types.NamespacedName]client.Object,
	namespace string,
	name string,
) {
	pageNavigation := &kdexv1alpha1.MicroFrontEndPageNavigation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kdexv1alpha1.MicroFrontEndPageNavigationSpec{
			Content: "<h1>Hello, from up north!</h1>",
		},
	}
	Expect(k8sClient.Create(ctx, pageNavigation)).To(Succeed())

	resourcesToDelete[types.NamespacedName{
		Name:      pageNavigation.Name,
		Namespace: namespace,
	}] = &kdexv1alpha1.MicroFrontEndPageNavigation{}
}
