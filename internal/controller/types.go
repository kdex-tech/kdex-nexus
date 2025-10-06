package controller

import (
	"encoding/base64"
	"fmt"

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
	Secure   bool     `json:"secure"`
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
	if c.Secure {
		return "https://" + c.Host
	} else {
		return "http://" + c.Host
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
