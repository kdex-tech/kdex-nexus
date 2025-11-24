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
	"kdex.dev/crds/base"
)

var _ = Describe("KDexPageNavigation Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageNavigation{
				KDexObject: base.KDexObject{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
				},
				Spec: kdexv1alpha1.KDexPageNavigationSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageNavigation{}, true)
		})

		It("should successfully reconcile after template becomes valid html", func() {
			addOrUpdatePageNavigation(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageNavigation{
					KDexObject: base.KDexObject{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: namespace,
						},
					},
					Spec: kdexv1alpha1.KDexPageNavigationSpec{
						Content: "<h1>Hello, World!</h1",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageNavigation{}, false)

			addOrUpdatePageNavigation(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageNavigation{
					KDexObject: base.KDexObject{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: namespace,
						},
					},
					Spec: kdexv1alpha1.KDexPageNavigationSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageNavigation{}, true)
		})

		It("should successfully reconcile after script library becomes available", func() {
			addOrUpdatePageNavigation(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageNavigation{
					KDexObject: base.KDexObject{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resourceName,
							Namespace: namespace,
						},
					},
					Spec: kdexv1alpha1.KDexPageNavigationSpec{
						Content: "<h1>Hello, World!</h1>",
						ScriptLibraryRef: &corev1.LocalObjectReference{
							Name: "non-existent-script-library",
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageNavigation{}, false)

			addOrUpdateScriptLibrary(
				ctx, k8sClient,
				kdexv1alpha1.KDexScriptLibrary{
					KDexObject: base.KDexObject{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "non-existent-script-library",
							Namespace: namespace,
						},
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
				&kdexv1alpha1.KDexPageNavigation{}, true)
		})
	})
})
