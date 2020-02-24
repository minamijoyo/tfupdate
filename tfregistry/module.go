package tfregistry

import (
	"context"
	"fmt"
)

const (
	// moduleV1Service is a sub path of module v1 service endpoint.
	// The service discovery protocol is not implemented for now.
	// https://www.terraform.io/docs/internals/remote-service-discovery.html
	//
	// Include slashes for later implementation of service discovery.
	// curl https://registry.terraform.io/.well-known/terraform.json
	// {"modules.v1":"/v1/modules/","providers.v1":"/v1/providers/"}
	moduleV1Service = "/v1/modules/"
)

// ModuleV1API is an interface for the module v1 service.
type ModuleV1API interface {
	// ModuleLatestForProvider returns the latest version of a module for a single provider.
	// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
	ModuleLatestForProvider(ctx context.Context, req *ModuleLatestForProviderRequest) (*ModuleLatestForProviderResponse, error)

	// ModuleListVersionsForProvider is the primary endpoint for resolving module sources, returning the available versions for a given fully-qualified module.
	// https://www.terraform.io/docs/registry/api.html#list-available-versions-for-a-specific-module
	ModuleListVersionsForProvider(ctx context.Context, req *ModuleListVersionsForProviderRequest) (*ModuleListVersionsForProviderResponse, error)
}

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

// ModuleListVersionsForProviderRequest is a request parameter for ModuleListVersionsForProvider().
// https://www.terraform.io/docs/registry/api.html#list-available-versions-for-a-specific-module
type ModuleListVersionsForProviderRequest struct {
	// Namespace is a user name which owns the module.
	Namespace string `json:"namespace"`
	// Name is a name of the module.
	Name string `json:"name"`
	// Provider is a name of the provider.
	Provider string `json:"provider"`
}

// ModuleListVersionsForProviderResponse is a response data for ModuleListVersionsForProvider().
// There are other response fields, but we define only those we need here.
type ModuleListVersionsForProviderResponse struct {
	// Version is the latest version of the module for a specific provider.
	Modules []*ModuleProviderVersions `json:"modules"`
}

// ModuleProviderVersions is a set of meta data for a module.
type ModuleProviderVersions struct {
	// Versions is a list of available versions of the module for a specific provider.
	Versions []*ModuleVersion `json:"versions"`
}

// ModuleVersion is a set of meta data of a single version.
type ModuleVersion struct {
	// Source is a
	Source string `json:"source"`
	// Version is a version of the module for a specific provider.
	Version string `json:"version"`
}

// ModuleListVersionsForProvider is the primary endpoint for resolving module sources, returning the available versions for a given fully-qualified module.
// https://www.terraform.io/docs/registry/api.html#list-available-versions-for-a-specific-module
func (c *Client) ModuleListVersionsForProvider(ctx context.Context, req *ModuleListVersionsForProviderRequest) (*ModuleListVersionsForProviderResponse, error) {
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

	var res ModuleListVersionsForProviderResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
