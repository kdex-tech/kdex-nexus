package render

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"kdex.dev/app-server/internal/menu"
)

type Renderer struct {
	Context      context.Context
	Date         time.Time
	FootScript   string
	HeadScript   string
	Lang         string
	MenuEntries  map[string]menu.MenuEntry
	Meta         string
	Organization string
	Stylesheet   string
}

func (r *Renderer) RenderAll(
	page Page,
	navigations map[string]string,
	header string,
	footer string,
) (string, error) {
	date := r.Date
	if date.IsZero() {
		date = time.Now()
	}

	templateData := TemplateData{
		Values: Values{
			Date:         date,
			FootScript:   template.HTML(r.FootScript),
			HeadScript:   template.HTML(r.HeadScript),
			Lang:         r.Lang,
			MenuEntries:  r.MenuEntries,
			Meta:         template.HTML(r.Meta),
			Organization: r.Organization,
			Stylesheet:   template.HTML(r.Stylesheet),
			Title:        page.Label,
		},
	}

	headerOutput, err := r.RenderOne(fmt.Sprintf("%s-header", page.TemplateName), header, templateData)
	if err != nil {
		return "", err
	}

	templateData.Values.Header = template.HTML(headerOutput)

	footerOutput, err := r.RenderOne(fmt.Sprintf("%s-footer", page.TemplateName), footer, templateData)
	if err != nil {
		return "", err
	}

	templateData.Values.Footer = template.HTML(footerOutput)

	navigationsOutput := make(map[string]template.HTML)
	for name, content := range navigations {
		currentNavigationOutput, err := r.RenderOne(fmt.Sprintf("%s-navigation-%s", page.TemplateName, name), content, templateData)
		if err != nil {
			return "", err
		}
		navigationsOutput[name] = template.HTML(currentNavigationOutput)
	}

	templateData.Values.Navigation = navigationsOutput

	contentOutputs := make(map[string]template.HTML)
	for _, contentEntry := range page.ContentEntries {
		var currentTemplate string
		var currentOutput string

		if contentEntry.AppRef == nil {
			currentTemplate = contentEntry.RawHTML
		} else {
			currentTemplate = fmt.Sprintf(`
				<%s
					data-date="{{.Values.Date.Format "2006-01-02"}}"
				>
				</%s>
			`, contentEntry.CustomElementName, contentEntry.CustomElementName)
		}

		currentOutput, err = r.RenderOne(fmt.Sprintf("%s-content-%s", page.TemplateName, contentEntry.Slot), currentTemplate, templateData)
		if err != nil {
			return "", err
		}

		contentOutputs[contentEntry.Slot] = template.HTML(currentOutput)
	}

	templateData.Values.Content = contentOutputs

	return r.RenderOne(page.TemplateName, page.TemplateContent, templateData)
}

func (r *Renderer) RenderOne(templateName string, templateContent string, data any) (string, error) {
	instance, err := template.New(templateName).Parse(templateContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := instance.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
