package render

import (
	"html/template"
	"time"

	"kdex.dev/app-server/internal/menu"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

type Page struct {
	ContentEntries  []kdexv1alpha1.ContentEntry
	Footer          string
	Header          string
	Label           string
	Navigations     map[string]string
	TemplateContent string
	TemplateName    string
}

type TemplateData struct {
	Values Values
}

type Values struct {
	Content      map[string]template.HTML
	Date         time.Time
	Footer       template.HTML
	FootScript   template.HTML
	Header       template.HTML
	HeadScript   template.HTML
	Lang         string
	Meta         template.HTML
	MenuEntries  map[string]menu.MenuEntry
	Navigation   map[string]template.HTML
	Organization string
	Title        string
	Stylesheet   template.HTML
}
