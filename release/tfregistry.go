package release

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

// TFRegistryAPI is an interface which calls Terraform Registry API.
// This abstraction layer is needed for testing with mock.
type TFRegistryAPI interface {
	// ModuleLatestForProvider returns the latest version of a module for a single provider.
	// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
	ModuleLatestForProvider(ctx context.Context, req *tfregistry.ModuleLatestForProviderRequest) (*tfregistry.ModuleLatestForProviderResponse, error)

	// ProviderLatest returns the latest version of a provider.
	// This relies on a currently undocumented providers API endpoint which behaves exactly like the equivalent documented modules API endpoint.
	// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
	ProviderLatest(ctx context.Context, req *tfregistry.ProviderLatestRequest) (*tfregistry.ProviderLatestResponse, error)
}

// TFRegistryConfig is a set of configurations for TFRegistryModuleRelease and TFRegistryProviderRelease.
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

// ModuleLatestForProvider returns the latest version of a module for a single provider.
// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
func (c *TFRegistryClient) ModuleLatestForProvider(ctx context.Context, req *tfregistry.ModuleLatestForProviderRequest) (*tfregistry.ModuleLatestForProviderResponse, error) {
	return c.client.ModuleLatestForProvider(ctx, req)
}

// TFRegistryModuleRelease is a release implementation which provides version information with TFRegistryModule Release.
type TFRegistryModuleRelease struct {
	// api is an instance of TFRegistryAPI interface.
	// It can be replaced for testing.
	api TFRegistryAPI

	// namespace is a user name which owns the module.
	namespace string

	// name is a name of the module.
	name string

	// provider is a name of the provider.
	provider string
}

var _ Release = (*TFRegistryModuleRelease)(nil)

// NewTFRegistryModuleRelease is a factory method which returns an TFRegistryModuleRelease instance.
func NewTFRegistryModuleRelease(source string, config TFRegistryConfig) (Release, error) {
	s := strings.SplitN(source, "/", 3)
	if len(s) != 3 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	// If config.api is not set, create a default TFRegistryClient
	var api TFRegistryAPI
	if config.api == nil {
		var err error
		api, err = NewTFRegistryClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		api = config.api
	}

	return &TFRegistryModuleRelease{
		api:       api,
		namespace: s[0],
		name:      s[1],
		provider:  s[2],
	}, nil
}

// ListReleases returns a list of unsorted all releases including pre-release.
func (r *TFRegistryModuleRelease) ListReleases(ctx context.Context) ([]string, error) {
	req := &tfregistry.ModuleLatestForProviderRequest{
		Namespace: r.namespace,
		Name:      r.name,
		Provider:  r.provider,
	}
	// Hard to guess from the name, the response of ModuleLatestForProvider API contains
	// not only the latest version, but also a list of available versions.
	release, err := r.api.ModuleLatestForProvider(ctx, req)

	if err != nil {
		return nil, fmt.Errorf("failed to get a list of versions for %s/%s/%s: %s", r.namespace, r.name, r.provider, err)
	}

	return release.Versions, nil
}

// ProviderLatest returns the latest version of a provider.
// This relies on a currently undocumented providers API endpoint which behaves exactly like the equivalent documented modules API endpoint.
// https://www.terraform.io/docs/registry/api.html#latest-version-for-a-specific-module-provider
func (c *TFRegistryClient) ProviderLatest(ctx context.Context, req *tfregistry.ProviderLatestRequest) (*tfregistry.ProviderLatestResponse, error) {
	return c.client.ProviderLatest(ctx, req)
}

// TFRegistryProviderRelease is a release implementation which provides version information with TFRegistryProvider Release.
type TFRegistryProviderRelease struct {
	// api is an instance of TFRegistryAPI interface.
	// It can be replaced for testing.
	api TFRegistryAPI

	// namespace is the name of a namespace, unique on a particular hostname, that can contain one or more providers that are somehow related. On the public Terraform Registry the "namespace" represents the organization that is packaging and distributing the provider.
	namespace string

	// providerType is the provider type, like "azurerm", "aws", "google", "dns", etc. A provider type is unique within a particular hostname and namespace.
	providerType string
}

var _ Release = (*TFRegistryProviderRelease)(nil)

// NewTFRegistryProviderRelease is a factory method which returns an TFRegistryProviderRelease instance.
func NewTFRegistryProviderRelease(source string, config TFRegistryConfig) (Release, error) {
	s := strings.SplitN(source, "/", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	// If config.api is not set, create a default TFRegistryClient
	var api TFRegistryAPI
	if config.api == nil {
		var err error
		api, err = NewTFRegistryClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		api = config.api
	}

	return &TFRegistryProviderRelease{
		api:          api,
		namespace:    s[0],
		providerType: s[1],
	}, nil
}

// ListReleases returns a list of unsorted all releases including pre-release.
func (r *TFRegistryProviderRelease) ListReleases(ctx context.Context) ([]string, error) {
	req := &tfregistry.ProviderLatestRequest{
		Namespace: r.namespace,
		Type:      r.providerType,
	}
	// Hard to guess from the name, the response of ProviderLatest API contains
	// not only the latest version, but also a list of available versions.
	release, err := r.api.ProviderLatest(ctx, req)

	if err != nil {
		return nil, fmt.Errorf("failed to get a list of versions for %s/%s: %s", r.namespace, r.providerType, err)
	}

	return release.Versions, nil
}
