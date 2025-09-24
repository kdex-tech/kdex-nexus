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
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("MicroFrontEndPageArchetype Controller", func() {
	Context("When reconciling a resource with missing default header reference", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		microfrontendpagearchetype := &kdexv1alpha1.MicroFrontEndPageArchetype{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind MicroFrontendPageArchetype")
			err := k8sClient.Get(ctx, typeNamespacedName, microfrontendpagearchetype)
			if err != nil && errors.IsNotFound(err) {
				resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
						DefaultHeaderRef: &corev1.LocalObjectReference{
							Name: "non-existent-header",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance MicroFrontendPageArchetype")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should not successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &MicroFrontEndPageArchetypeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, microfrontendpagearchetype)
			Expect(err).NotTo(HaveOccurred())

			Expect(
				apimeta.IsStatusConditionFalse(
					microfrontendpagearchetype.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
				),
			).To(BeTrue())
		})
	})

	Context("When reconciling a resource with only content", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		microfrontendpagearchetype := &kdexv1alpha1.MicroFrontEndPageArchetype{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind MicroFrontendPageArchetype")
			err := k8sClient.Get(ctx, typeNamespacedName, microfrontendpagearchetype)
			if err != nil && errors.IsNotFound(err) {
				resource := &kdexv1alpha1.MicroFrontEndPageArchetype{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kdexv1alpha1.MicroFrontEndPageArchetypeSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &kdexv1alpha1.MicroFrontEndPageArchetype{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance MicroFrontendPageArchetype")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &MicroFrontEndPageArchetypeReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, microfrontendpagearchetype)
			Expect(err).NotTo(HaveOccurred())

			Expect(
				apimeta.IsStatusConditionTrue(
					microfrontendpagearchetype.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
				),
			).To(BeTrue())
		})
	})
})
