package page

import (
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResolvedContentEntry struct {
	AppObj            client.Object
	Attributes        map[string]string
	Content           string
	CustomElementName string
	PackageReference  *kdexv1alpha1.PackageReference
	Scripts           []kdexv1alpha1.ScriptDef
	Slot              string
}

type ResolvedNavigation struct {
	Generation int64
	Name       string
	Spec       *kdexv1alpha1.KDexPageNavigationSpec
}
