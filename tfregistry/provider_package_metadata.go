package tfregistry

import (
	"context"
	"fmt"
	"log"
)

// ProviderPackageMetadataRequest is a request parameter for ProviderPackageMetadata().
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
type ProviderPackageMetadataRequest struct {
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

// ProviderPackageMetadataResponse is a response data for ProviderPackageMetadata().
// There are other response fields, but we define only those we need here.
type ProviderPackageMetadataResponse struct {
	// (required): the filename for this provider's zip archive as recorded in the "shasums" document, so that Terraform CLI can determine which of the given checksums should be used for this specific package.
	Filename string `json:"filename"`
	// (required): a URL from which Terraform can retrieve the provider's zip archive. If this is a relative URL then it will be resolved relative to the URL that returned the containing JSON object.
	DownloadURL string `json:"download_url"`
	// (required): the SHA256 checksum for this provider's zip archive as recorded in the shasums document.
	SHASum string `json:"shasum"`
	// (required): a URL from which Terraform can retrieve a text document recording expected SHA256 checksums for this package and possibly other packages for the same provider version on other platforms.
	SHASumsURL string `json:"shasums_url"`
}

// ProviderPackageMetadata returns a package metadata of a provider.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
func (c *Client) ProviderPackageMetadata(ctx context.Context, req *ProviderPackageMetadataRequest) (*ProviderPackageMetadataResponse, error) {
	if len(req.Namespace) == 0 {
		return nil, fmt.Errorf("Invalid request. Namespace is required. req = %#v", req)
	}
	if len(req.Type) == 0 {
		return nil, fmt.Errorf("Invalid request. Type is required. req = %#v", req)
	}
	if len(req.Version) == 0 {
		return nil, fmt.Errorf("Invalid request. Version is required. req = %#v", req)
	}
	if len(req.OS) == 0 {
		return nil, fmt.Errorf("Invalid request. OS is required. req = %#v", req)
	}
	if len(req.Arch) == 0 {
		return nil, fmt.Errorf("Invalid request. Arch is required. req = %#v", req)
	}

	subPath := fmt.Sprintf("%s%s/%s/%s/download/%s/%s", providerV1Service, req.Namespace, req.Type, req.Version, req.OS, req.Arch)

	httpRequest, err := c.newRequest(ctx, "GET", subPath, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Client.ProviderPackageMetadata: GET %s", httpRequest.URL)
	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to HTTP Request: err = %s, req = %#v", err, httpRequest)
	}

	if httpResponse.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP Status Code: %d", httpResponse.StatusCode)
	}

	var res ProviderPackageMetadataResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
