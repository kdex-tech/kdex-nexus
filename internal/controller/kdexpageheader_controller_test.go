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

var _ = Describe("KDexPageHeader Controller", func() {
	Context("When reconciling a resource", func() {
		const namespace = "default"
		const resourceName = "test-resource"

		ctx := context.Background()

		AfterEach(func() {
			cleanupResources(namespace)
		})

		It("should successfully reconcile the resource", func() {
			resource := &kdexv1alpha1.KDexPageHeader{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: kdexv1alpha1.KDexPageHeaderSpec{
					Content: "<h1>Hello, World!</h1>",
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageHeader{}, true)
		})

		It("should successfully reconcile after template becomes valid html", func() {
			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "<h1>Hello, World!</h1",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageHeader{}, false)

			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "<h1>Hello, World!</h1>",
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageHeader{}, true)
		})

		It("should successfully reconcile after script library becomes available", func() {
			addOrUpdatePageHeader(
				ctx, k8sClient,
				kdexv1alpha1.KDexPageHeader{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: namespace,
					},
					Spec: kdexv1alpha1.KDexPageHeaderSpec{
						Content: "<h1>Hello, World!</h1>",
						ScriptLibraryRef: &kdexv1alpha1.KDexObjectReference{
							Kind: "KDexScriptLibrary",
							Name: "non-existent-script-library",
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageHeader{}, false)

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
								Script: "console.log('test');",
							},
						},
					},
				},
			)

			assertResourceReady(
				ctx, k8sClient, resourceName, namespace,
				&kdexv1alpha1.KDexPageHeader{}, true)
		})
	})
})
