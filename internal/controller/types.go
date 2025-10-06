package controller

import (
	"encoding/base64"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AuthData struct {
	Password string `json:"password"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

type ClientObjectWithConditions struct {
	client.Object
	Conditions *[]metav1.Condition
}

type NPMRegistryConfiguration struct {
	AuthData AuthData `json:"authData"`
	Host     string   `json:"host"`
	InSecure bool     `json:"insecure"`
}

func NPMRegistryConfigurationNew(secret *corev1.Secret) *NPMRegistryConfiguration {
	if secret == nil {
		return &NPMRegistryConfiguration{
			AuthData: AuthData{
				Password: "",
				Token:    "",
				Username: "",
			},
			Host:     "registry.npmjs.org",
			InSecure: false,
		}
	}

	return &NPMRegistryConfiguration{
		AuthData: AuthData{
			Password: string(secret.Data["password"]),
			Token:    string(secret.Data["token"]),
			Username: string(secret.Data["username"]),
		},
		Host:     string(secret.Labels["kdex.dev/npm-server-address"]),
		InSecure: string(secret.Labels["kdex.dev/npm-server-insecure"]) == "true",
	}
}

func (c *NPMRegistryConfiguration) EncodeAuthorization() string {
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

func (c *NPMRegistryConfiguration) GetAddress() string {
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
