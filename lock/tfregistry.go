package lock

import (
	"context"
	"fmt"
	"net/url"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

// TFRegistryAPI is an interface which calls Terraform Registry API.
// This abstraction layer is needed for testing with mock.
type TFRegistryAPI interface {
	// ProviderPackageMetadata returns a package metadata of a provider.
	// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
	ProviderPackageMetadata(ctx context.Context, req *tfregistry.ProviderPackageMetadataRequest) (*tfregistry.ProviderPackageMetadataResponse, error)
}

// TFRegistryConfig is a set of configurations for ProviderDownloaderClient.
type TFRegistryConfig struct {
	// api is an instance of TFRegistryAPI interface.
	// It can be replaced for testing.
	api TFRegistryAPI

	// BaseURL is a URL for Terraform Registry API requests.
	// Defaults to the public Terraform Registry API.
	// This looks like the Terraform Cloud support, but currently for testing purposes only.
	// The Terraform Cloud is not supported yet.
	// BaseURL should always be specified with a trailing slash.
	BaseURL string
}

// TFRegistryClient is a real TFRegistryAPI implementation.
type TFRegistryClient struct {
	client *tfregistry.Client
}

var _ TFRegistryAPI = (*TFRegistryClient)(nil)

// NewTFRegistryClient returns a real TFRegistryClient instance.
func NewTFRegistryClient(config TFRegistryConfig) (*TFRegistryClient, error) {
	c := tfregistry.NewClient(nil)

	if len(config.BaseURL) != 0 {
		baseURL, err := url.Parse(config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tfregistry base url: %s", err)
		}
		c.BaseURL = baseURL
	}

	return &TFRegistryClient{
		client: c,
	}, nil
}

// ProviderPackageMetadata returns a package metadata of a provider.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
func (c *TFRegistryClient) ProviderPackageMetadata(ctx context.Context, req *tfregistry.ProviderPackageMetadataRequest) (*tfregistry.ProviderPackageMetadataResponse, error) {
	return c.client.ProviderPackageMetadata(ctx, req)
}
