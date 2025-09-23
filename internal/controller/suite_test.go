/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/client-go/rest"
	kdexv1alpha1 "kdex.dev/crds/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)

func addRemoteCRD(paths *[]string, tempDir string, url string) {
	crdPath, err := downloadCRD(url, tempDir)
	if err != nil {
		panic(err)
	}

	*paths = append(*paths, crdPath)
}

func downloadCRD(url, tempDir string) (string, error) {
	httpClient := &http.Client{}
	response, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download CRD from %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download CRD from %s: status code %d", url, response.StatusCode)
	}

	crdContent, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read CRD content from %s: %w", url, err)
	}

	fileName := filepath.Base(url)
	filePath := filepath.Join(tempDir, fileName)
	err = os.WriteFile(filePath, crdContent, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write CRD to file %s: %w", filePath, err)
	}

	return filePath, nil
}

func getCRDModuleVersion() string {
	return "v0.3.2"
}

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{}, // No local CRDs initially
		ErrorIfCRDPathMissing: true,
	}

	tempDir, err := os.MkdirTemp("", "crd")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendapps.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendhosts.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendpagearchetypes.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendpagebindings.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendpagefooters.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendpageheaders.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendpagenavigations.yaml", getCRDModuleVersion()))
	addRemoteCRD(&testEnv.CRDDirectoryPaths, tempDir, fmt.Sprintf("https://raw.githubusercontent.com/kdex-tech/kdex-crds/refs/tags/%s/config/crd/bases/kdex.dev_microfrontendrenderpages.yaml", getCRDModuleVersion()))

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	err = kdexv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
