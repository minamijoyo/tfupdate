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
	// ProviderLatest returns the latest version of a provider.
	// This relies on a currently undocumented providers API endpoint which behaves exactly like the equivalent documented modules API endpoint.
	// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
	ProviderLatest(ctx context.Context, req *ProviderLatestRequest) (*ProviderLatestResponse, error)

	// ProviderPackageMetadata returns a package metadata of a provider.
	// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
	ProviderPackageMetadata(ctx context.Context, req *ProviderPackageMetadataRequest) (*ProviderPackageMetadataResponse, error)
}
