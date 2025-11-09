package controller

import (
	"bytes"
	"fmt"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/render"
)

func validateScripts(scriptReference *kdexv1alpha1.KDexScriptLibrarySpec) error {
	renderer := render.Renderer{}

	td := render.DefaultTemplateData()

	// validate head scripts
	var buffer bytes.Buffer
	separator := ""

	if scriptReference.PackageReference != nil {
		if scriptReference.PackageReference.Name == "" {
			return fmt.Errorf("package reference name is required")
		}

		buffer.WriteString(scriptReference.PackageReference.ToScriptTag())
		separator = "\n"
	}

	for _, script := range scriptReference.Scripts {
		output := script.ToScriptTag(false)
		if output != "" {
			buffer.WriteString(separator)
			separator = "\n"
			buffer.WriteString(output)
		}
	}

	if _, err := renderer.RenderOne("head-scripts", buffer.String(), td); err != nil {
		return fmt.Errorf("failed to validate head scripts: %w", err)
	}

	// validate foot scripts
	buffer = bytes.Buffer{}
	separator = ""

	for _, script := range scriptReference.Scripts {
		output := script.ToScriptTag(true)
		if output != "" {
			buffer.WriteString(separator)
			separator = "\n"
			buffer.WriteString(output)
		}
	}

	if _, err := renderer.RenderOne("validate-script-library", buffer.String(), td); err != nil {
		return fmt.Errorf("failed to validate foot scripts: %w", err)
	}

	return nil
}
