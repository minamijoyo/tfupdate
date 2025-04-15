package release

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

// TFRegistryAPI is an interface which calls Terraform Registry API.
// This works for both Terraform and OpenTofu registries.
// This abstraction layer is needed for testing with mock.
type TFRegistryAPI interface {
	// ListModuleVersions returns all versions of a module for a single provider.
	ListModuleVersions(ctx context.Context, req *tfregistry.ListModuleVersionsRequest) (*tfregistry.ListModuleVersionsResponse, error)

	// ListProviderVersions returns all versions of a provider.
	ListProviderVersions(ctx context.Context, req *tfregistry.ListProviderVersionsRequest) (*tfregistry.ListProviderVersionsResponse, error)
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

// NewDefaultTerraformRegistryConfig returns a TFRegistryConfig with the default
// BaseURL for the public Terraform Registry.
func NewDefaultTerraformRegistryConfig() TFRegistryConfig {
	return TFRegistryConfig{
		BaseURL: "https://registry.terraform.io/",
	}
}

// NewDefaultOpenTofuRegistryConfig returns a TFRegistryConfig with the default
// BaseURL for the public OpenTofu Registry.
func NewDefaultOpenTofuRegistryConfig() TFRegistryConfig {
	return TFRegistryConfig{
		BaseURL: "https://registry.opentofu.org/",
	}
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

// ListModuleVersions returns all versions of a module for a single provider.
func (c *TFRegistryClient) ListModuleVersions(ctx context.Context, req *tfregistry.ListModuleVersionsRequest) (*tfregistry.ListModuleVersionsResponse, error) {
	return c.client.ListModuleVersions(ctx, req)
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
	req := &tfregistry.ListModuleVersionsRequest{
		Namespace: r.namespace,
		Name:      r.name,
		Provider:  r.provider,
	}

	response, err := r.api.ListModuleVersions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of versions for %s/%s/%s: %s", r.namespace, r.name, r.provider, err)
	}

	// Extract versions from the response
	if len(response.Modules) == 0 {
		return []string{}, nil
	}

	versions := []string{}
	for _, version := range response.Modules[0].Versions {
		versions = append(versions, version.Version)
	}

	return versions, nil
}

// ListProviderVersions returns all versions of a provider.
func (c *TFRegistryClient) ListProviderVersions(ctx context.Context, req *tfregistry.ListProviderVersionsRequest) (*tfregistry.ListProviderVersionsResponse, error) {
	return c.client.ListProviderVersions(ctx, req)
}

// TFRegistryProviderRelease is a release implementation which provides version information with TFRegistryProvider Release.
type TFRegistryProviderRelease struct {
	// api is an instance of TFRegistryAPI interface.
	// It can be replaced for testing.
	api TFRegistryAPI

	// The user or organization the provider is owned by.
	namespace string

	// The type name of the provider.
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
	req := &tfregistry.ListProviderVersionsRequest{
		Namespace: r.namespace,
		Type:      r.providerType,
	}

	response, err := r.api.ListProviderVersions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get a list of versions for %s/%s: %s", r.namespace, r.providerType, err)
	}

	// Extract versions from the response
	versions := []string{}
	for _, version := range response.Versions {
		versions = append(versions, version.Version)
	}

	return versions, nil
}
