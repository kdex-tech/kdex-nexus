package render

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"kdex.dev/app-server/internal/menu"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

func TestRenderOne(t *testing.T) {
	r := &Renderer{}
	templateContent := "Hello, {{.Name}}!"
	data := struct{ Name string }{Name: "World"}
	expected := "Hello, World!"
	actual, err := r.RenderOne("test", templateContent, data)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestRenderOne_InvalidTemplate(t *testing.T) {
	r := &Renderer{}
	templateContent := "Hello, {{.Invalid}}!"
	data := struct{ Name string }{Name: "World"}
	_, err := r.RenderOne("test", templateContent, data)
	assert.Error(t, err)
}

func TestRenderAll(t *testing.T) {
	testDate, _ := time.Parse("2006-01-02", "2025-09-20")
	r := &Renderer{
		Context:      context.Background(),
		Date:         testDate,
		FootScript:   "<script>foot</script>",
		HeadScript:   "<script>head</script>",
		Lang:         "en",
		MenuEntries:  map[string]menu.MenuEntry{"home": {Path: "/"}},
		Meta:         `<meta name="description" content="test">`,
		Organization: "Test Inc.",
		Stylesheet:   "<style>body{}</style>",
	}

	page := Page{
		Label:        "Test Page",
		TemplateName: "main",
		TemplateContent: `
<html>
	<head>
		<title>{{.Values.Title}}</title>
		{{.Values.Meta}}
		{{.Values.HeadScript}}
		{{.Values.Stylesheet}}
	</head>
	<body>
		<header>{{.Values.Header}}</header>
		<nav>{{range $key, $value := .Values.Navigation}}
			{{$key}}: {{$value}}
		{{end}}</nav>
		<main>{{range $key, $value := .Values.Content}}
			<div id="slot-{{$key}}">{{$value}}</div>
		{{end}}</main>
		<footer>{{.Values.Footer}}</footer>
		{{.Values.FootScript}}
	</body>
</html>`,
		ContentEntries: []kdexv1alpha1.ContentEntry{
			{
				Slot:    "main",
				RawHTML: "<h1>Welcome</h1>",
			},
			{
				Slot: "sidebar",
				AppRef: &corev1.LocalObjectReference{
					Name: "my-app",
				},
				CustomElementName: "my-app-element",
			},
		},
	}

	navigations := map[string]string{
		"main": "main-nav",
	}

	header := "Page Header"
	footer := "Page Footer"

	actual, err := r.RenderAll(page, navigations, header, footer)
	assert.NoError(t, err)

	assert.Contains(t, actual, "<title>Test Page</title>")
	assert.Contains(t, actual, r.Meta)
	assert.Contains(t, actual, r.HeadScript)
	assert.Contains(t, actual, r.Stylesheet)
	assert.Contains(t, actual, "Page Header")
	assert.Contains(t, actual, "main: main-nav")
	assert.Contains(t, actual, "<h1>Welcome</h1>")
	assert.Contains(t, actual, "<my-app-element")
	assert.Contains(t, actual, "2025-09-20")
	assert.Contains(t, actual, "Page Footer")
	assert.Contains(t, actual, r.FootScript)
}

func TestRenderAll_InvalidHeaderTemplate(t *testing.T) {
	r := &Renderer{}
	page := Page{
		TemplateName: "main",
	}
	_, err := r.RenderAll(page, nil, "{{.Invalid}}", "")
	assert.Error(t, err)
}

func TestRenderAll_InvalidFooterTemplate(t *testing.T) {
	r := &Renderer{}
	page := Page{
		TemplateName: "main",
	}
	_, err := r.RenderAll(page, nil, "", "{{.Invalid}}")
	assert.Error(t, err)
}

func TestRenderAll_InvalidNavigationTemplate(t *testing.T) {
	r := &Renderer{}
	page := Page{
		TemplateName: "main",
	}
	navigations := map[string]string{
		"main": "{{.Invalid}}",
	}
	_, err := r.RenderAll(page, navigations, "", "")
	assert.Error(t, err)
}

func TestRenderAll_InvalidContentTemplate(t *testing.T) {
	r := &Renderer{}
	page := Page{
		TemplateName: "main",
		ContentEntries: []kdexv1alpha1.ContentEntry{
			{
				Slot:    "main",
				RawHTML: "{{.Invalid}}",
			},
		},
	}
	_, err := r.RenderAll(page, nil, "", "")
	assert.Error(t, err)
}

func TestRenderAll_InvalidMainTemplate(t *testing.T) {
	r := &Renderer{}
	page := Page{
		TemplateName:    "main",
		TemplateContent: "{{.Invalid}}",
	}
	_, err := r.RenderAll(page, nil, "", "")
	assert.Error(t, err)
}
