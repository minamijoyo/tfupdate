package tfregistry

import (
	"context"
	"fmt"
)

// ModuleLatestForProviderRequest is a request parameter for ModuleLatestForProvider().
// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
type ModuleLatestForProviderRequest struct {
	// Namespace is a user name which owns the module.
	Namespace string `json:"namespace"`
	// Name is a name of the module.
	Name string `json:"name"`
	// Provider is a name of the provider.
	Provider string `json:"provider"`
}

// ModuleLatestForProviderResponse is a response data for ModuleLatestForProvider().
// There are other response fields, but we define only those we need here.
type ModuleLatestForProviderResponse struct {
	// Version is the latest version of the module for a specific provider.
	Version string `json:"version"`
	// Versions is a list of available versions.
	Versions []string `json:"versions"`
}

// ModuleLatestForProvider returns the latest version of a module for a single provider.
// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
func (c *Client) ModuleLatestForProvider(ctx context.Context, req *ModuleLatestForProviderRequest) (*ModuleLatestForProviderResponse, error) {
	if len(req.Namespace) == 0 {
		return nil, fmt.Errorf("Invalid request. Namespace is required. req = %#v", req)
	}
	if len(req.Name) == 0 {
		return nil, fmt.Errorf("Invalid request. Name is required. req = %#v", req)
	}
	if len(req.Provider) == 0 {
		return nil, fmt.Errorf("Invalid request. Provider is required. req = %#v", req)
	}

	subPath := fmt.Sprintf("%s%s/%s/%s", moduleV1Service, req.Namespace, req.Name, req.Provider)

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

	var res ModuleLatestForProviderResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
