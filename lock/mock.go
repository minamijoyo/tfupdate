package lock

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"golang.org/x/exp/slices"
)

// newMockServer returns a new mock server for testing.
func newMockServer() (*http.ServeMux, *url.URL) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mockServerURL, _ := url.Parse(server.URL)
	return mux, mockServerURL
}

// newTestClient returns a new client for testing.
func newTestClient(mockServerURL *url.URL, config TFRegistryConfig) *ProviderDownloaderClient {
	config.BaseURL = mockServerURL.String()
	c, _ := NewProviderDownloaderClient(config)
	return c
}

// newMockZipData returns a new zip format data for testing.
func newMockZipData(filename string, contents string) ([]byte, error) {
	// create a zip file in memory
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// create a file in the zip file
	w, err := zw.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create a file in zip: err = %s", err)
	}
	_, err = w.Write([]byte(contents))
	if err != nil {
		return nil, fmt.Errorf("failed to write contents to a file: err = %s", err)
	}

	// zip
	err = zw.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to flush a zip file: err = %s", err)
	}

	return buf.Bytes(), nil
}

// newMockShaSumsData returns a new shaSumsData for testing.
// To ensure that the dummy data can be re-used in other test cases, the
// function really creates a zip file in memory and calculates its sha256sum.
func newMockShaSumsData(name string, version string, platforms []string) ([]byte, error) {
	// terraform-provider-dummy_v3.2.1_x5
	filename := fmt.Sprintf("terraform-provider-%s_v%s_x5", name, version)
	lines := []string{}
	for _, platform := range platforms {
		// dummy_3.2.1_darwin_arm64
		contents := fmt.Sprintf("%s_%s_%s", name, version, platform)

		// create a zip file in memory.
		zipData, err := newMockZipData(filename, contents)
		if err != nil {
			return nil, fmt.Errorf("failed to create a zip file in memory: err = %s", err)
		}
		zh := sha256sumAsHexString(zipData)
		zipFilename := "terraform-provider-" + contents + ".zip"
		line := fmt.Sprintf("%s  %s", zh, zipFilename)
		lines = append(lines, line)
	}

	slices.Sort(lines)
	document := strings.Join(lines, "\n")
	return []byte(document), nil
}

// newMockProviderDownloadResponse returns a new ProviderDownloadResponse for testing.
// Note that some parameters are hard-coded to simplify the caller.
func newMockProviderDownloadResponse(platform string) (*ProviderDownloadResponse, error) {
	// create a zip file in memory.
	zipData, err := newMockZipData("terraform-provider-dummy_v3.2.1_x5", "dummy_3.2.1_"+platform)
	if err != nil {
		return nil, fmt.Errorf("failed to create a zip file in memory: err = %s", err)
	}
	// create a valid dummy shaSumsData.
	platforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64", "windows_amd64"}
	shaSumsData, err := newMockShaSumsData("dummy", "3.2.1", platforms)
	if err != nil {
		return nil, fmt.Errorf("failed to create a shaSumsData: err = %s", err)
	}
	filename := fmt.Sprintf("terraform-provider-dummy_3.2.1_%s.zip", platform)
	return &ProviderDownloadResponse{
		filename:    filename,
		zipData:     zipData,
		shaSumsData: shaSumsData,
	}, nil
}

// newMockProviderDownloadResponses returns a new list of ProviderDownloadResponse for testing.
func newMockProviderDownloadResponses(platforms []string) ([]*ProviderDownloadResponse, error) {
	responses := []*ProviderDownloadResponse{}
	for _, platform := range platforms {
		res, err := newMockProviderDownloadResponse(platform)
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}

	return responses, nil
}

// NewMockIndex does not call the real API but returns preset mock provider version metadata.
func NewMockIndex(pvs []*ProviderVersion) Index {
	i := &index{
		providers: make(map[string]*providerIndex),
		papi:      nil,
	}
	for _, pv := range pvs {
		pi, ok := i.providers[pv.address]
		if !ok {
			pi = newProviderIndex(pv.address, i.papi)
			i.providers[pv.address] = pi
		}
		pi.versions[pv.version] = pv
	}

	return i
}

// NewMockProviderVersion returns a mocked ProviderVersion for testing.
// This is actually a setter to all private fields, but should not be used
// except for generating test data from outside the package.
func NewMockProviderVersion(address string, version string, platforms []string, h1Hashes map[string]string, zhHashes map[string]string) *ProviderVersion {
	return &ProviderVersion{
		address:   address,
		version:   version,
		platforms: platforms,
		h1Hashes:  h1Hashes,
		zhHashes:  zhHashes,
	}
}
