package validate

import (
	htmltemplate "html/template"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
	"k8s.io/apimachinery/pkg/api/resource"
	"kdex.dev/crds/render"
	"kdex.dev/crds/template"
)

var renderer render.Renderer
var templateData template.TemplateData

func init() {
	translations := catalog.NewBuilder()
	translations.SetString(language.English, "name", "Name")
	translations.SetString(language.French, "name", "Nom")

	messagePrinter := message.NewPrinter(
		language.English,
		message.Catalog(translations),
	)

	templateData = template.TemplateData{
		Footer:       `<p>footer</p>`,
		FootScript:   `<script type="text/javascript"></script>`,
		Header:       `<p>header</p>`,
		HeadScript:   `<script type="text/javascript"></script>`,
		Language:     "en",
		Languages:    []string{"en", "fr"},
		LastModified: time.Now(),
		LeftToRight:  true,
		Meta:         `<meta charset="UTF-8">`,
		Organization: "KDex Tech Inc.",
		PageMap: map[string]*template.PageEntry{
			"One": {
				Href:   "/one",
				Icon:   "one",
				Label:  "One",
				Name:   "one",
				Weight: resource.MustParse("0"),
			},
			"Two": {
				Href:   "/two",
				Icon:   "two",
				Label:  "Two",
				Name:   "two",
				Weight: resource.MustParse("1"),
			},
			"Three": {
				Children: &map[string]*template.PageEntry{
					"Four": {
						Href:   "/four",
						Icon:   "four",
						Label:  "Four",
						Name:   "four",
						Weight: resource.MustParse("0"),
					},
				},
				Href:   "/three",
				Icon:   "three",
				Label:  "Three",
				Name:   "three",
				Weight: resource.MustParse("3"),
			},
		},
		Stylesheet: `<style></style>`,
		Title:      "name",
	}

	contents := map[string]htmltemplate.HTML{}
	contents["main"] = htmltemplate.HTML("<p>content</p>")
	templateData.Content = contents

	navigations := map[string]htmltemplate.HTML{}
	navigations["main"] = htmltemplate.HTML("<p>navigation</p>")
	templateData.Navigation = navigations

	renderer = render.Renderer{
		MessagePrinter: messagePrinter,
	}
}

func TemplateContent(
	name string, content string,
) error {
	_, err := renderer.RenderOne(name, content, templateData)

	return err
}
