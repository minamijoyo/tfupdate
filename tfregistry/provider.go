package tfregistry

import (
	"context"
)

const (
	// providerV1Service is a sub path of provider v1 service endpoint.
	// The service discovery protocol is not implemented for now.
	// https://www.terraform.io/docs/internals/provider-registry-protocol.html#service-discovery
	//
	// Include slashes for later implementation of service discovery.
	// curl https://registry.terraform.io/.well-known/terraform.json
	// {"modules.v1":"/v1/modules/","providers.v1":"/v1/providers/"}
	providerV1Service = "/v1/providers/"
)

// ProviderV1API is an interface for the provider v1 service.
type ProviderV1API interface {
	// ListProviderVersions returns all versions of a provider.
	// This works for both Terraform and OpenTofu registries.
	// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#list-available-versions
	// https://opentofu.org/docs/internals/provider-registry-protocol/#list-available-versions
	ListProviderVersions(ctx context.Context, req *ListProviderVersionsRequest) (*ListProviderVersionsResponse, error)

	// ProviderPackageMetadata returns a package metadata of a provider.
	// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
	ProviderPackageMetadata(ctx context.Context, req *ProviderPackageMetadataRequest) (*ProviderPackageMetadataResponse, error)
}
