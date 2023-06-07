package lock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

// PackageDownloaderAPI is an interface for downloading provider package.
// Provider packages are downloaded from the HashiCorp release server,
// therefore we distinct this API from the Terraform Registry API.
// The API specification is not documented.
type ProviderDownloaderAPI interface {
	// ProviderDownload downloads a provider package.
	ProviderDownload(ctx context.Context, req *ProviderDownloadRequest) (*ProviderDownloadResponse, error)
}

// ProviderDownloaderClient implements the ProviderDownloaderAPI interface
type ProviderDownloaderClient struct {
	// api is an instance of TFRegistryAPI interface.
	// It can be replaced for testing.
	api TFRegistryAPI

	// httpClient is a http client which communicates with the ProviderDownloaderAPI.
	httpClient *http.Client
}

// ProviderDownloaderClient is a factory method which returns a ProviderDownloaderClient instance.
func NewProviderDownloaderClient(config TFRegistryConfig) (*ProviderDownloaderClient, error) {
	// If config.api is not set, create a default TFRegistryClient
	var api TFRegistryAPI
	if config.api == nil {
		var err error
		api, err = NewTFRegistryClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		api = config.api
	}

	return &ProviderDownloaderClient{
		api:        api,
		httpClient: &http.Client{},
	}, nil
}

// ProviderDownloadRequest is a request type for ProviderDownload.
type ProviderDownloadRequest struct {
	// (required): the namespace portion of the address of the requested provider.
	Namespace string `json:"namespace"`
	// (required): the type portion of the address of the requested provider.
	Type string `json:"type"`
	// (required): the version selected to download.
	Version string `json:"version"`
	// (required): a keyword identifying the operating system that the returned package should be compatible with, like "linux" or "darwin".
	OS string `json:"os"`
	// (required): a keyword identifying the CPU architecture that the returned package should be compatible with, like "amd64" or "arm".
	Arch string `json:"arch"`
}

// ProviderDownloadResponse is a response type for ProviderDownload.
type ProviderDownloadResponse struct {
	// zipData is the raw byte sequence of the provider package.
	zipData []byte

	// SHA256Sum is a check sum for zipData in the sha256 sum function.
	SHA256Sum string
}

// ProviderDownload downloads a provider package.
func (c *ProviderDownloaderClient) ProviderDownload(ctx context.Context, req *ProviderDownloadRequest) (*ProviderDownloadResponse, error) {
	metadataReq := &tfregistry.ProviderPackageMetadataRequest{
		Namespace: req.Namespace,
		Type:      req.Type,
		Version:   req.Version,
		OS:        req.OS,
		Arch:      req.Arch,
	}

	metadataRes, err := c.api.ProviderPackageMetadata(ctx, metadataReq)
	if err != nil {
		return nil, err
	}

	downloadURL := metadataRes.DownloadURL
	zipData, err := c.download(ctx, downloadURL)
	if err != nil {
		return nil, err
	}

	sha256sum := metadataRes.SHASum
	err = validateSHA256Sum(zipData, metadataRes.SHASum)
	if err != nil {
		return nil, err
	}

	ret := &ProviderDownloadResponse{
		zipData:   zipData,
		SHA256Sum: sha256sum,
	}

	return ret, nil
}

// download is a helper function that downloads a package from a given url.
func (c *ProviderDownloaderClient) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build http request: err = %s, url = %s", err, url)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request: err = %s, url = %s", err, url)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %s: %s", res.Status, url)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: err = %s, url = %s", err, url)
	}

	return data, nil
}

// validateSHA256Sum calculates the sha256 sum of the given byte sequence and
// checks whether it matches the expected hash value.
// The hash value is specified as a hexadecimal string.
func validateSHA256Sum(b []byte, sha256sum string) error {
	got := sha256sumAsHexString(b)
	if got != sha256sum {
		return fmt.Errorf("checksum missmatch error. got = %s, expected = %s", got, sha256sum)
	}

	return nil
}

// sha256sumAsHexString calculates the sha256 sum of the given byte sequence and
// returns it as a hexadecimal string.
func sha256sumAsHexString(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
