package scaffolder

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"kdex.dev/crds/configuration"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
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
	{{ range $path, $info := .Paths }}

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

{{ range $path, $info := .Paths }}

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

func Scaffold(funcObj *kdexv1alpha1.KDexFunction, config configuration.NexusConfiguration) (stub *kdexv1alpha1.StubDetails, err error) {
	ctx := context.Background()
	data := FunctionData{
		Name:  funcObj.Name,
		Path:  funcObj.Spec.API.BasePath,
		Paths: funcObj.Spec.API.Paths,
	}

	// Use a temporary directory for generation
	basePath, err := os.MkdirTemp("", "kdex-scaffold-"+funcObj.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(basePath) }()

	filePath := filepath.Join(basePath, "main.go")
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	tmpl, err := template.New("go-func").Parse(goTemplate)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	_ = f.Close()

	// Create a simple go.mod
	goModPath := filepath.Join(basePath, "go.mod")
	goModContent := fmt.Sprintf("module %s\n\ngo 1.25.0\n", funcObj.Name)
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %w", err)
	}

	// Prepare local file store for ORAS
	store, err := file.New(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Add files to the store and get their descriptors
	files := []string{"main.go", "go.mod"}
	descs := make([]ocispec.Descriptor, 0, len(files))
	for _, name := range files {
		desc, err := store.Add(ctx, name, "", "")
		if err != nil {
			return nil, fmt.Errorf("failed to add file %s to store: %w", name, err)
		}
		descs = append(descs, desc)
	}

	// Pack the files into a manifest
	// We use application/vnd.kdex.function.source.v1+json as the artifact type
	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_0, "application/vnd.kdex.function.source.v1+json", oras.PackManifestOptions{
		Layers: descs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack files: %w", err)
	}

	// Define target image name: registry/namespace/name:latest
	imageName := fmt.Sprintf("%s/%s/%s", config.DefaultImageRegistry.Host, funcObj.Namespace, funcObj.Name)
	repo, err := remote.NewRepository(imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote repository: %w", err)
	}

	if config.DefaultImageRegistry.InSecure {
		repo.PlainHTTP = true
	}

	// Set up authentication
	if config.DefaultImageRegistry.AuthData.Username != "" {
		repo.Client = &auth.Client{
			Client: http.DefaultClient,
			Credential: auth.StaticCredential(config.DefaultImageRegistry.Host, auth.Credential{
				Username: config.DefaultImageRegistry.AuthData.Username,
				Password: config.DefaultImageRegistry.AuthData.Password,
			}),
		}
	} else if config.DefaultImageRegistry.AuthData.Token != "" {
		repo.Client = &auth.Client{
			Client: http.DefaultClient,
			Credential: auth.StaticCredential(config.DefaultImageRegistry.Host, auth.Credential{
				AccessToken: config.DefaultImageRegistry.AuthData.Token,
			}),
		}
	}

	// Push the manifest and its blobs to the remote repository
	_, err = oras.Copy(ctx, store, manifestDesc.Digest.String(), repo, "latest", oras.CopyOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to push to remote: %w", err)
	}

	fullImageName := fmt.Sprintf("%s:latest", imageName)
	return &kdexv1alpha1.StubDetails{
		FilePath:    "main.go",
		Language:    "go",
		SourceImage: fullImageName,
	}, nil
}
