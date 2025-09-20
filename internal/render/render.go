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
	templateData := TemplateData{
		Values: Values{
			Date:         time.Now(),
			FootScript:   r.FootScript,
			HeadScript:   r.HeadScript,
			Lang:         r.Lang,
			MenuEntries:  r.MenuEntries,
			Meta:         r.Meta,
			Organization: r.Organization,
			Stylesheet:   r.Stylesheet,
			Title:        page.Label,
		},
	}

	headerOutput, err := r.RenderOne(fmt.Sprintf("%s-header", page.TemplateName), header, templateData)
	if err != nil {
		return "", err
	}

	templateData.Values.Header = &headerOutput

	footerOutput, err := r.RenderOne(fmt.Sprintf("%s-footer", page.TemplateName), footer, templateData)
	if err != nil {
		return "", err
	}

	templateData.Values.Footer = &footerOutput

	navigationsOutput := make(map[string]string)
	for name, content := range navigations {
		currentNavigationOutput, err := r.RenderOne(fmt.Sprintf("%s-navigation-%s", page.TemplateName, name), content, templateData)
		if err != nil {
			return "", err
		}
		navigationsOutput[name] = currentNavigationOutput
	}

	templateData.Values.Navigation = &navigationsOutput

	contentOutputs := make(map[string]string)
	for _, contentEntry := range page.ContentEntries {
		var data any
		var template string
		var contentOutput string

		if contentEntry.AppRef == nil {
			template = contentEntry.RawHTML
			data = templateData
		} else {
			template = `
				<{{.CustomElementName}} 
					data-date="{{.Values.Date.Format('2006-01-02')}}"
				>
				</{{.CustomElementName}}>
			`
			data = AppData{
				CustomElementName: contentEntry.CustomElementName,
				Values:            templateData.Values,
			}
		}

		contentOutput, err = r.RenderOne(fmt.Sprintf("%s-content-%s", page.TemplateName, contentEntry.Slot), template, data)
		if err != nil {
			return "", err
		}

		contentOutputs[contentEntry.Slot] = contentOutput
	}

	templateData.Values.Content = &contentOutputs

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
