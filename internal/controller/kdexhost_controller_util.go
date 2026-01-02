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
	"maps"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *KDexHostReconciler) createOrUpdateInternalUtilityPage(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
	utilityPageSpec kdexv1alpha1.KDexUtilityPageSpec,
	pageType kdexv1alpha1.KDexUtilityPageType,
) (*corev1.LocalObjectReference, error) {
	log := ctrl.LoggerFrom(ctx)

	name := fmt.Sprintf("%s-%s", host.Name, strings.ToLower(string(pageType)))
	internalUtilityPage := &kdexv1alpha1.KDexInternalUtilityPage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: host.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, internalUtilityPage, func() error {
		if internalUtilityPage.CreationTimestamp.IsZero() {
			internalUtilityPage.Annotations = make(map[string]string)
			maps.Copy(internalUtilityPage.Annotations, host.Annotations)
			internalUtilityPage.Labels = make(map[string]string)
			maps.Copy(internalUtilityPage.Labels, host.Labels)

			internalUtilityPage.Labels["app.kubernetes.io/name"] = kdexWeb
			internalUtilityPage.Labels["kdex.dev/instance"] = host.Name
			internalUtilityPage.Labels["kdex.dev/utility-page-type"] = string(pageType)
		}

		internalUtilityPage.Spec.KDexUtilityPageSpec = utilityPageSpec
		internalUtilityPage.Spec.HostRef = corev1.LocalObjectReference{Name: host.Name}

		return ctrl.SetControllerReference(host, internalUtilityPage, r.Scheme)
	})

	log.V(1).Info(
		"createOrUpdateInternalUtilityPage",
		"name", name,
		"type", pageType,
		"op", op,
		"err", err,
	)

	if err != nil {
		return nil, err
	}

	return &corev1.LocalObjectReference{Name: name}, nil
}

func (r *KDexHostReconciler) resolveUtilityPages(
	ctx context.Context,
	host *kdexv1alpha1.KDexHost,
) ([]kdexv1alpha1.KDexObjectReference, *corev1.LocalObjectReference, *corev1.LocalObjectReference, *corev1.LocalObjectReference, error) {
	requiredBackends := []kdexv1alpha1.KDexObjectReference{}
	refs := map[kdexv1alpha1.KDexUtilityPageType]*corev1.LocalObjectReference{}

	types := []kdexv1alpha1.KDexUtilityPageType{
		kdexv1alpha1.AnnouncementUtilityPageType,
		kdexv1alpha1.ErrorUtilityPageType,
		kdexv1alpha1.LoginUtilityPageType,
	}

	for _, pageType := range types {
		var ref *kdexv1alpha1.KDexObjectReference
		if host.Spec.UtilityPages != nil {
			switch pageType {
			case kdexv1alpha1.AnnouncementUtilityPageType:
				ref = host.Spec.UtilityPages.AnnouncementRef
			case kdexv1alpha1.ErrorUtilityPageType:
				ref = host.Spec.UtilityPages.ErrorRef
			case kdexv1alpha1.LoginUtilityPageType:
				ref = host.Spec.UtilityPages.LoginRef
			}
		}

		directRef := true

		if ref == nil {
			directRef = false
			ref = &kdexv1alpha1.KDexObjectReference{
				Kind: "KDexClusterUtilityPage",
				Name: fmt.Sprintf("kdex-default-%s", strings.ToLower(string(pageType))),
			}
		}

		resolvedObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, ref, r.RequeueDelay)
		if shouldReturn {
			if directRef && err == nil {
				err = fmt.Errorf("utility page %s not found", ref.Name)
			}
			return nil, nil, nil, nil, err
		}

		if resolvedObj != nil {
			var spec kdexv1alpha1.KDexUtilityPageSpec
			switch v := resolvedObj.(type) {
			case *kdexv1alpha1.KDexUtilityPage:
				spec = v.Spec
			case *kdexv1alpha1.KDexClusterUtilityPage:
				spec = v.Spec
			}

			// Validate Type matches
			if spec.Type != pageType {
				return nil, nil, nil, nil, fmt.Errorf("utility page type %s does not match requested type %s", spec.Type, pageType)
			}

			internalRef, err := r.createOrUpdateInternalUtilityPage(ctx, host, spec, pageType)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			refs[pageType] = internalRef

			host.Status.Attributes["utilitypage."+strings.ToLower(string(pageType))+".generation"] = fmt.Sprintf("%d", resolvedObj.GetGeneration())

			// === Collect Backends ===

			// 1. Archetype
			archetypeObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, &spec.PageArchetypeRef, r.RequeueDelay)
			if shouldReturn {
				return nil, nil, nil, nil, err
			}
			if archetypeObj != nil {
				CollectBackend(&requiredBackends, archetypeObj)

				var archetypeSpec kdexv1alpha1.KDexPageArchetypeSpec
				switch v := archetypeObj.(type) {
				case *kdexv1alpha1.KDexPageArchetype:
					archetypeSpec = v.Spec
				case *kdexv1alpha1.KDexClusterPageArchetype:
					archetypeSpec = v.Spec
				}

				// Archetype ScriptLibrary
				archetypeSLObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, archetypeSpec.ScriptLibraryRef, r.RequeueDelay)
				if shouldReturn {
					return nil, nil, nil, nil, err
				}
				if archetypeSLObj != nil {
					CollectBackend(&requiredBackends, archetypeSLObj)
				}
			}

			// 2. Content Entries
			for _, content := range spec.ContentEntries {
				if content.AppRef != nil {
					appObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, content.AppRef, r.RequeueDelay)
					if shouldReturn {
						return nil, nil, nil, nil, err
					}
					if appObj != nil {
						CollectBackend(&requiredBackends, appObj)
					}
				}
			}

			// 3. Main Navigation
			navRef := spec.OverrideMainNavigationRef
			if navRef != nil {
				navObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, navRef, r.RequeueDelay)
				if shouldReturn {
					return nil, nil, nil, nil, err
				}
				if navObj != nil {
					var navSpec kdexv1alpha1.KDexPageNavigationSpec
					switch v := navObj.(type) {
					case *kdexv1alpha1.KDexPageNavigation:
						navSpec = v.Spec
					case *kdexv1alpha1.KDexClusterPageNavigation:
						navSpec = v.Spec
					}

					navSLObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, navSpec.ScriptLibraryRef, r.RequeueDelay)
					if shouldReturn {
						return nil, nil, nil, nil, err
					}
					if navSLObj != nil {
						CollectBackend(&requiredBackends, navSLObj)
					}
				}
			}

			// 4. Header
			headerRef := spec.OverrideHeaderRef
			if headerRef != nil {
				headerObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, headerRef, r.RequeueDelay)
				if shouldReturn {
					return nil, nil, nil, nil, err
				}
				if headerObj != nil {
					var headerSpec kdexv1alpha1.KDexPageHeaderSpec
					switch v := headerObj.(type) {
					case *kdexv1alpha1.KDexPageHeader:
						headerSpec = v.Spec
					case *kdexv1alpha1.KDexClusterPageHeader:
						headerSpec = v.Spec
					}

					headerSLObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, headerSpec.ScriptLibraryRef, r.RequeueDelay)
					if shouldReturn {
						return nil, nil, nil, nil, err
					}
					if headerSLObj != nil {
						CollectBackend(&requiredBackends, headerSLObj)
					}
				}
			}

			// 5. Footer
			footerRef := spec.OverrideFooterRef
			if footerRef != nil {
				footerObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, footerRef, r.RequeueDelay)
				if shouldReturn {
					return nil, nil, nil, nil, err
				}
				if footerObj != nil {
					var footerSpec kdexv1alpha1.KDexPageFooterSpec
					switch v := footerObj.(type) {
					case *kdexv1alpha1.KDexPageFooter:
						footerSpec = v.Spec
					case *kdexv1alpha1.KDexClusterPageFooter:
						footerSpec = v.Spec
					}

					footerSLObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, footerSpec.ScriptLibraryRef, r.RequeueDelay)
					if shouldReturn {
						return nil, nil, nil, nil, err
					}
					if footerSLObj != nil {
						CollectBackend(&requiredBackends, footerSLObj)
					}
				}
			}

			// 6. Page Script Library
			pageSLObj, shouldReturn, _, err := ResolveKDexObjectReference(ctx, r.Client, host, &host.Status.Conditions, spec.ScriptLibraryRef, r.RequeueDelay)
			if shouldReturn {
				return nil, nil, nil, nil, err
			}
			if pageSLObj != nil {
				CollectBackend(&requiredBackends, pageSLObj)
			}
		}
	}

	return requiredBackends, refs[kdexv1alpha1.AnnouncementUtilityPageType], refs[kdexv1alpha1.ErrorUtilityPageType], refs[kdexv1alpha1.LoginUtilityPageType], nil
}
