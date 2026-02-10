package internal

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	SHARED_VOLUME = "shared-volume"
	WORKDIR       = "/shared"
)

var KPackImageGVK = schema.GroupVersionKind{
	Group:   "kpack.io",
	Version: "v1alpha2",
	Kind:    "Image",
}
