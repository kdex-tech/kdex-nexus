package scaffolder

import (
	"errors"
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
var methods = []string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE"}

func main() {
	{{ $path, $info := range .Paths }}

	{{ if $info.Connect }}
	http.HandleFunc("CONNECT {{ $path }}", ConnectHandler)
	{{ end }}

	{{ if $info.Delete }}
	http.HandleFunc("DELETE {{ $path }}", DeleteHandler)
	{{ end }}

	{{ if $info.Get }}
	http.HandleFunc("GET {{ $path }}", GetHandler)
	{{ end }}

	{{ if $info.Head }}
	http.HandleFunc("HEAD {{ $path }}", HeadHandler)
	{{ end }}

	{{ if $info.Options }}
	http.HandleFunc("OPTIONS {{ $path }}", OptionsHandler)
	{{ end }}

	{{ if $info.Patch }}
	http.HandleFunc("PATCH {{ $path }}", PatchHandler)
	{{ end }}

	{{ if $info.Post }}
	http.HandleFunc("POST {{ $path }}", PostHandler)
	{{ end }}

	{{ if $info.Put }}
	http.HandleFunc("PUT {{ $path }}", PutHandler)
	{{ end }}

	{{ if $info.Trace }}
	http.HandleFunc("TRACE {{ $path }}", TraceHandler)
	{{ end }}

	{{ end }}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting server on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}

{{ $path, $info := range .Paths }}

{{ if $info.Connect }}
func ConnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "CONNECT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "Hello from KDex Function: {{ .Name }}\n")
	fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
	fmt.Fprintf(w, "Method: %s\n", r.Method)
}
{{ end }}

{{ if $info.Delete }}
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

{{ if $info.Get }}
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

{{ if $info.Post }}
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

{{ if $info.Put }}
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

{{ if $info.Patch }}
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

{{ if $info.Options }}
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

{{ if $info.Head }}
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

{{ end }}
`

type FunctionData struct {
	Name  string
	Path  string
	Paths map[string]kdexv1alpha1.PathItem
}

func Scaffold(funcObj *kdexv1alpha1.KDexFunction) (stub *kdexv1alpha1.StubDetails, err error) {
	data := FunctionData{
		Name:  funcObj.Name,
		Path:  funcObj.Spec.API.BasePath,
		Paths: funcObj.Spec.API.Paths,
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
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

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
