package tfregistry

import (
	"context"
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
}
