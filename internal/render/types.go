package render

import (
	"time"

	"kdex.dev/app-server/internal/menu"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

type AppData struct {
	CustomElementName string
	Values            Values
}

type Page struct {
	ContentEntries  []kdexv1alpha1.ContentEntry
	Label           string
	TemplateContent string
	TemplateName    string
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
	MenuEntries  map[string]menu.MenuEntry
	Navigation   *map[string]string
	Organization string
	Title        string
	Stylesheet   string
}
