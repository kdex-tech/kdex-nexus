package controller

import (
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AppData struct {
	CustomElementName string
	Values            Values
}

type ClientObjectWithConditions struct {
	client.Object
	Conditions []metav1.Condition
}

type MenuEntry struct {
	Children *map[string]MenuEntry
	Icon     string
	Path     string
	Weight   resource.Quantity
}

type TemplateData struct {
	Values Values
}

type Values struct {
	Content      *map[string]string
	Date         time.Time
	Footer       *string
	FootScript   string
	Header       *string
	HeadScript   string
	Lang         string
	Meta         string
	MenuEntries  map[string]MenuEntry
	Navigation   *map[string]string
	Organization string
	Title        string
	Stylesheet   string
}
