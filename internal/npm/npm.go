package npm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
)

type Registry interface {
	ValidatePackage(packageName string, packageVersion string) error
}

type RegistryImpl struct {
	Config *RegistryConfiguration
	Error  func(err error, msg string, keysAndValues ...any)
}

func NewRegistry(secret *corev1.Secret, error func(err error, msg string, keysAndValues ...any)) Registry {
	return &RegistryImpl{
		Config: RegistryConfigurationNew(secret),
		Error:  error,
	}
}

func (r *RegistryImpl) GetPackageInfo(packageName string) (*PackageInfo, error) {
	packageURL := fmt.Sprintf("%s/%s", r.Config.GetAddress(), packageName)

	req, err := http.NewRequest("GET", packageURL, nil)
	if err != nil {
		return nil, err
	}

	authorization := r.Config.EncodeAuthorization()
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	req.Header.Set("Accept", "application/vnd.npm.formats+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.Error(err, "failed to close response body")
		}
	}()

	fmt.Println("Response Status:", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package not found: %s", packageURL)
	}

	packageInfo := &PackageInfo{}

	var body []byte
	if body, err = io.ReadAll(resp.Body); err == nil {
		if err = json.Unmarshal(body, &packageInfo); err != nil {
			return nil, err
		}
	}

	return packageInfo, nil
}

func (r *RegistryImpl) ValidatePackage(packageName string, packageVersion string) error {
	packageInfo, err := r.GetPackageInfo(packageName)

	if err != nil {
		return err
	}

	versionPackageJSON, ok := packageInfo.Versions[packageVersion]

	if !ok {
		return fmt.Errorf("version of package not found: %s", packageName+"@"+packageVersion)
	}

	return versionPackageJSON.HasESModule()
}
