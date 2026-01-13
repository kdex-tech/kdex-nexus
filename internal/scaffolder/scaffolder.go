package scaffolder

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
)

const goTemplate = `package main

import (
	"fmt"
	"net/http"
	"os"
)

// get, put, post, delete, options, head, patch, trace
var methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

func main() {
	{{ if .Spec.Get }}
	http.HandleFunc("GET {{ .Path }}", GetHandler)
	{{ end }}
	{{ if .Spec.Post }}
	http.HandleFunc("POST {{ .Path }}", PostHandler)
	{{ end }}
	{{ if .Spec.Put }}
	http.HandleFunc("PUT {{ .Path }}", PutHandler)
	{{ end }}
	{{ if .Spec.Delete }}
	http.HandleFunc("DELETE {{ .Path }}", DeleteHandler)
	{{ end }}
	{{ if .Spec.Patch }}
	http.HandleFunc("PATCH {{ .Path }}", PatchHandler)
	{{ end }}
	{{ if .Spec.Options }}
	http.HandleFunc("OPTIONS {{ .Path }}", OptionsHandler)
	{{ end }}
	{{ if .Spec.Head }}
	http.HandleFunc("HEAD {{ .Path }}", HeadHandler)
	{{ end }}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting server on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}

{{ if .Spec.Get }}
func GetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if .Spec.Post }}
func PostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if .Spec.Put }}
func PutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if .Spec.Delete }}
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if .Spec.Patch }}
func PatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PATCH" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if .Spec.Options }}
func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "OPTIONS" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if .Spec.Head }}
func HeadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "HEAD" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}
`

type FunctionData struct {
	Name string
	Path string
	Spec kdexv1alpha1.KDexOpenAPIInternal
}

func Scaffold(funcObj *kdexv1alpha1.KDexFunction) (*kdexv1alpha1.StubDetails, error) {
	data := FunctionData{
		Name: funcObj.Name,
		Path: funcObj.Spec.API.Path,
		Spec: funcObj.Spec.API.KDexOpenAPIInternal,
	}

	// For simulation, we'll write to a "generated" directory in the workspace if SourceRepository is empty
	// or points to a local directory.
	basePath := funcObj.Spec.Metadata.SourceRepository
	if basePath == "" {
		// Default to a temporary location for simulation
		basePath = filepath.Join("/tmp", "kdex-functions", funcObj.Name)
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(basePath, "main.go")
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("go-func").Parse(goTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &kdexv1alpha1.StubDetails{
		Language: "go",
		FilePath: filePath,
	}, nil
}
