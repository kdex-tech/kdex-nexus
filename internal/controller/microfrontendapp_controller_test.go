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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("MicroFrontEndApp Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}
		microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}

		AfterEach(func() {
			resource := &kdexv1alpha1.MicroFrontEndApp{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance MicroFrontEndApp")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile a valid resource", func() {
			By("Creating the reconciler")
			controllerReconciler := &MicroFrontEndAppReconciler{
				Client:       k8sClient,
				Scheme:       k8sClient.Scheme(),
				RequeueDelay: 0,
			}

			By("Creating the resource")
			resource := &kdexv1alpha1.MicroFrontEndApp{
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
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)
			Expect(err).NotTo(HaveOccurred())

			Expect(
				apimeta.IsStatusConditionTrue(
					microfrontendapp.Status.Conditions, string(kdexv1alpha1.ConditionTypeReady),
				),
			).To(BeFalse())
		})
	})
})
