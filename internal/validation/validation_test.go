package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

func Test_ValidateScriptLibrary(t *testing.T) {
	tests := []struct {
		name    string
		spec    *kdexv1alpha1.KDexScriptLibrarySpec
		wantErr bool
	}{
		{
			name: "basic script",
			spec: &kdexv1alpha1.KDexScriptLibrarySpec{
				Scripts: []kdexv1alpha1.ScriptDef{
					{
						Script: `console.log('test');`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "basic script fail",
			spec: &kdexv1alpha1.KDexScriptLibrarySpec{
				Scripts: []kdexv1alpha1.ScriptDef{
					{
						Script: `console.log('test`,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "basic package reference",
			spec: &kdexv1alpha1.KDexScriptLibrarySpec{
				PackageReference: &kdexv1alpha1.PackageReference{
					Name:    "@foo/bar",
					Version: "1.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "basic package reference fail",
			spec: &kdexv1alpha1.KDexScriptLibrarySpec{
				PackageReference: &kdexv1alpha1.PackageReference{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScriptLibrary(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
