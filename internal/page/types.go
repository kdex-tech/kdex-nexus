package page

import (
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

type ResolvedContentEntry struct {
	App               *kdexv1alpha1.KDexAppSpec
	AppName           string
	AppGeneration     string
	Content           string
	CustomElementName string
	Slot              string
}

type ResolvedNavigation struct {
	Generation int64
	Name       string
	Spec       *kdexv1alpha1.KDexPageNavigationSpec
}
