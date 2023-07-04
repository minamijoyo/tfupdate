package tfregistry

import (
	"context"
	"fmt"
)

// ProviderLatestRequest is a request parameter for ProviderLatest().
// This relies on a currently undocumented providers API endpoint which behaves exactly like the equivalent documented modules API endpoint.
// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
type ProviderLatestRequest struct {
	// Namespace is the name of a namespace, unique on a particular hostname, that can contain one or more providers that are somehow related. On the public Terraform Registry the "namespace" represents the organization that is packaging and distributing the provider.
	Namespace string `json:"namespace"`
	// Type is the provider type, like "azurerm", "aws", "google", "dns", etc. A provider type is unique within a particular hostname and namespace.
	Type string `json:"type"`
}

// ProviderLatestResponse is a response data for ProviderLatest().
// There are other response fields, but we define only those we need here.
type ProviderLatestResponse struct {
	// Version is the latest version of the provider.
	Version string `json:"version"`
	// Versions is a list of available versions.
	Versions []string `json:"versions"`
}

// ProviderLatest returns the latest version of a provider.
// This relies on a currently undocumented providers API endpoint which behaves exactly like the equivalent documented modules API endpoint.
// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
func (c *Client) ProviderLatest(ctx context.Context, req *ProviderLatestRequest) (*ProviderLatestResponse, error) {
	if len(req.Namespace) == 0 {
		return nil, fmt.Errorf("Invalid request. Namespace is required. req = %#v", req)
	}
	if len(req.Type) == 0 {
		return nil, fmt.Errorf("Invalid request. Type is required. req = %#v", req)
	}

	subPath := fmt.Sprintf("%s%s/%s", providerV1Service, req.Namespace, req.Type)

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

	var res ProviderLatestResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
