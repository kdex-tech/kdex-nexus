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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
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

		resourcesToDelete[types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}] = &kdexv1alpha1.MicroFrontEndPageBinding{}

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
		})

		It("with missing references should not succeed", NodeTimeout(time.Second*90), func(ctx SpecContext) {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}
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
			host := &kdexv1alpha1.MicroFrontEndHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-host",
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

			pageArchetype := &kdexv1alpha1.MicroFrontEndPageArchetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-page-archetype",
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

			check = func(g Gomega) {
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(
					apimeta.IsStatusConditionTrue(
						resource.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
					),
				).To(BeTrue())
			}

			Eventually(check).WithTimeout(30 * time.Second).Should(Succeed())
		})
	})
})
