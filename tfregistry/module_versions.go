package tfregistry

import (
	"context"
	"fmt"
)

// ListModuleVersionsRequest is a request parameter for the ListModuleVersions API.
type ListModuleVersionsRequest struct {
	// The user or organization the module is owned by.
	Namespace string `json:"namespace"`
	// The name of the module.
	Name string `json:"name"`
	// The name of the provider.
	Provider string `json:"provider"`
}

// ListModuleVersionsResponse is a response data for the ListModuleVersions API.
type ListModuleVersionsResponse struct {
	// Modules is an array containing module information.
	// The first element contains the requested module.
	Modules []ModuleVersions `json:"modules"`
}

// ModuleVersions represents version information for a module.
type ModuleVersions struct {
	// Versions is a list of available versions.
	Versions []ModuleVersion `json:"versions"`
}

// ModuleVersion represents a single version of a module.
type ModuleVersion struct {
	// Version is the version string.
	Version string `json:"version"`
}

// ListModuleVersions returns all versions of a module for a single provider.
// This works for both Terraform and OpenTofu registries.
func (c *Client) ListModuleVersions(ctx context.Context, req *ListModuleVersionsRequest) (*ListModuleVersionsResponse, error) {
	if len(req.Namespace) == 0 {
		return nil, fmt.Errorf("Invalid request. Namespace is required. req = %#v", req)
	}
	if len(req.Name) == 0 {
		return nil, fmt.Errorf("Invalid request. Name is required. req = %#v", req)
	}
	if len(req.Provider) == 0 {
		return nil, fmt.Errorf("Invalid request. Provider is required. req = %#v", req)
	}

	subPath := fmt.Sprintf("%s%s/%s/%s/versions", moduleV1Service, req.Namespace, req.Name, req.Provider)

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

	var res ListModuleVersionsResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
