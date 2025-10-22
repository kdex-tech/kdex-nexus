package customelement

import (
	"fmt"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

func ForApp(
	app kdexv1alpha1.MicroFrontEndApp,
	contentEntry kdexv1alpha1.ContentEntry,
) string {
	return fmt.Sprintf(`
			<%s
				data-app-generation="%d"
				data-app-name="%s"
				data-app-resource-version="%s"
				id="%s"
			>
			</%s>
		`,
		contentEntry.CustomElementName,
		app.Generation,
		app.Name,
		app.ResourceVersion,
		contentEntry.Slot,
		contentEntry.CustomElementName)
}
