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
	// ListModuleVersions returns all versions of a module for a single provider.
	// This works for both Terraform and OpenTofu registries.
	// https://developer.hashicorp.com/terraform/registry/api-docs#list-available-versions-for-a-specific-module
	// https://opentofu.org/docs/internals/module-registry-protocol/#list-available-versions-for-a-specific-module
	ListModuleVersions(ctx context.Context, req *ListModuleVersionsRequest) (*ListModuleVersionsResponse, error)
}
