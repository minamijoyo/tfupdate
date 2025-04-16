package tfregistry

import (
	"context"
	"fmt"
)

// ListProviderVersionsRequest is a request parameter for ListProviderVersions API.
type ListProviderVersionsRequest struct {
	// The user or organization the provider is owned by.
	Namespace string `json:"namespace"`
	// The type name of the provider.
	Type string `json:"type"`
}

// ListProviderVersionsResponse is a response data for ListProviderVersions API.
type ListProviderVersionsResponse struct {
	// Versions is a list of available versions.
	Versions []ProviderVersion `json:"versions"`
}

// ProviderVersion represents a single version of a provider.
type ProviderVersion struct {
	// Version is the version string.
	Version string `json:"version"`
	// Protocols is a list of supported protocol versions.
	Protocols []string `json:"protocols,omitempty"`
	// Platforms is a list of supported platforms.
	Platforms []ProviderPlatform `json:"platforms,omitempty"`
}

// ProviderPlatform represents a platform supported by a provider version.
type ProviderPlatform struct {
	// OS is the operating system.
	OS string `json:"os"`
	// Arch is the architecture.
	Arch string `json:"arch"`
}

// ListProviderVersions returns all versions of a provider.
// This works for both Terraform and OpenTofu registries.
func (c *Client) ListProviderVersions(ctx context.Context, req *ListProviderVersionsRequest) (*ListProviderVersionsResponse, error) {
	if len(req.Namespace) == 0 {
		return nil, fmt.Errorf("Invalid request. Namespace is required. req = %#v", req)
	}
	if len(req.Type) == 0 {
		return nil, fmt.Errorf("Invalid request. Type is required. req = %#v", req)
	}

	subPath := fmt.Sprintf("%s%s/%s/versions", providerV1Service, req.Namespace, req.Type)

	httpRequest, err := c.newRequest(ctx, "GET", subPath, nil)
	if err != nil {
		return nil, err
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to HTTP Request: err = %s, req = %#v", err, httpRequest)
	}

	if httpResponse.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP Status Code: %d", httpResponse.StatusCode)
	}

	var res ListProviderVersionsResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
