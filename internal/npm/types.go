package npm

import (
	"encoding/base64"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type AuthData struct {
	Password string `json:"password"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

type RegistryConfiguration struct {
	AuthData AuthData `json:"authData"`
	Host     string   `json:"host"`
	InSecure bool     `json:"insecure"`
}

func RegistryConfigurationNew(secret *corev1.Secret) *RegistryConfiguration {
	if secret == nil ||
		secret.Labels == nil ||
		secret.Labels["kdex.dev/npm-server-address"] == "" {

		return &RegistryConfiguration{
			AuthData: AuthData{
				Password: "",
				Token:    "",
				Username: "",
			},
			Host:     "registry.npmjs.org",
			InSecure: false,
		}
	}

	return &RegistryConfiguration{
		AuthData: AuthData{
			Password: string(secret.Data["password"]),
			Token:    string(secret.Data["token"]),
			Username: string(secret.Data["username"]),
		},
		Host:     secret.Labels["kdex.dev/npm-server-address"],
		InSecure: secret.Labels["kdex.dev/npm-server-insecure"] == "true",
	}
}

func (c *RegistryConfiguration) EncodeAuthorization() string {
	token := c.AuthData.Token
	if token != "" {
		return "Bearer " + token
	}

	if c.AuthData.Username != "" && c.AuthData.Password != "" {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(
			fmt.Sprintf("%s:%s", c.AuthData.Username, c.AuthData.Password)),
		)
	}

	return ""
}

func (c *RegistryConfiguration) GetAddress() string {
	if c.InSecure {
		return "http://" + c.Host
	} else {
		return "https://" + c.Host
	}
}

type PackageInfo struct {
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Versions map[string]PackageJSON `json:"versions"`
}

type PackageJSON struct {
	Author             string            `json:"author"`
	Browser            string            `json:"browser"`
	Bugs               interface{}       `json:"bugs"`
	BundleDependencies []string          `json:"bundleDependencies"`
	Dependencies       map[string]string `json:"dependencies"`
	Description        string            `json:"description"`
	DevDependencies    map[string]string `json:"devDependencies"`
	Dist               struct {
		Integrity string `json:"integrity"`
		Shasum    string `json:"shasum"`
		Tarball   string `json:"tarball"`
	} `json:"dist"`
	Exports              map[string]string `json:"exports"`
	Homepage             string            `json:"homepage"`
	Keywords             []string          `json:"keywords"`
	License              string            `json:"license"`
	Main                 string            `json:"main"`
	Name                 string            `json:"name"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	Private              bool              `json:"private"`
	Repository           interface{}       `json:"repository"`
	Scripts              map[string]string `json:"scripts"`
	Type                 string            `json:"type"`
	Version              string            `json:"version"`
}

func (p *PackageJSON) HasESModule() error {
	if p.Browser != "" {
		return nil
	}

	if p.Type == "module" {
		return nil
	}

	if p.Exports != nil {
		browser, ok := p.Exports["browser"]

		if ok && browser != "" {
			return nil
		}

		imp, ok := p.Exports["import"]

		if ok && imp != "" {
			return nil
		}
	}

	if strings.HasSuffix(p.Main, ".mjs") {
		return nil
	}

	return fmt.Errorf("package does not contain an ES module")
}
