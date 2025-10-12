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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/nexus/internal/npm"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type MockRegistry struct{}

func (m *MockRegistry) ValidatePackage(packageName string, packageVersion string) error {
	return nil
}

var _ = Describe("MicroFrontEndApp Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		resources := map[types.NamespacedName]client.Object{}

		resources[types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}] = &kdexv1alpha1.MicroFrontEndApp{}

		AfterEach(func() {
			By("Cleanup all the test resource instances")
			for name, resource := range resources {
				err := k8sClient.Get(ctx, name, resource)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should not successfully reconcile an invalid resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			assertCreateReconcilerForApp(
				typeNamespacedName,
				&kdexv1alpha1.MicroFrontEndApp{
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
					},
				},
				&MockRegistry{},
				true,
			)
		})

		It("should successfully reconcile a valid resource", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			assertCreateReconcilerForApp(
				typeNamespacedName,
				&kdexv1alpha1.MicroFrontEndApp{
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
				},
				&MockRegistry{},
				false,
			)

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)

				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionTrue),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal(string(kdexv1alpha1.ConditionReasonReconcileSuccess)),
				)
				g.Expect(
					condition.Message,
				).To(
					Equal("all references resolved successfully"),
				)
			}

			Eventually(check).Should(Succeed())
		})

		It("a resource with a invalid package reference should become failed", func() {
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: namespace,
			}

			assertCreateReconcilerForApp(
				typeNamespacedName,
				&kdexv1alpha1.MicroFrontEndApp{
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
							Name:    "my-scope/my-package",
							Version: "1.0.0",
						},
					},
				},
				&MockRegistry{},
				true,
			)

			check := func(g Gomega) {
				microfrontendapp := &kdexv1alpha1.MicroFrontEndApp{}
				err := k8sClient.Get(ctx, typeNamespacedName, microfrontendapp)
				g.Expect(err).NotTo(HaveOccurred())

				condition := apimeta.FindStatusCondition(
					microfrontendapp.Status.Conditions,
					string(kdexv1alpha1.ConditionTypeReady),
				)

				g.Expect(condition).ToNot(BeNil())
				g.Expect(
					condition.Status,
				).To(
					Equal(metav1.ConditionFalse),
				)
				g.Expect(
					condition.Reason,
				).To(
					Equal("PackageValidationFailed"),
				)
				g.Expect(
					condition.Message,
				).To(
					ContainSubstring("invalid package name, must be scoped with @scope/name:"),
				)
			}

			Eventually(check).Should(Succeed())
		})
	})
})

func assertCreateReconcilerForApp(
	typeNamespacedName types.NamespacedName,
	app *kdexv1alpha1.MicroFrontEndApp,
	registry npm.Registry,
	resultsInError bool,
) *MicroFrontEndAppReconciler {
	By("Creating the reconciler")

	controllerReconciler := &MicroFrontEndAppReconciler{
		Client: k8sClient,
		RegistryFactory: func(
			secret *corev1.Secret,
			error func(err error, msg string, keysAndValues ...any),
		) npm.Registry {
			return registry
		},
		RequeueDelay: 0,
		Scheme:       k8sClient.Scheme(),
	}

	By("Creating the resource")

	Expect(k8sClient.Create(ctx, app)).To(Succeed())

	By("Reconciling the created resource")

	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})

	if resultsInError {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}

	return controllerReconciler
}
