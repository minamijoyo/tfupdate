package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

// TFRegistryModuleRelease is a release implementation which provides version information with TFRegistryModule Release.
type TFRegistryModuleRelease struct {
	// api is an instance of tfregistry.API interface.
	// It can be replaced for testing.
	api tfregistry.API

	// namespace is a user name which owns the module.
	namespace string

	// name is a name of the module.
	name string

	// provider is a name of the provider.
	provider string
}

var _ Release = (*TFRegistryModuleRelease)(nil)

// NewTFRegistryModuleRelease is a factory method which returns an TFRegistryModuleRelease instance.
func NewTFRegistryModuleRelease(source string, config tfregistry.Config) (Release, error) {
	s := strings.SplitN(source, "/", 3)
	if len(s) != 3 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	client, err := tfregistry.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &TFRegistryModuleRelease{
		api:       client,
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

// TFRegistryProviderRelease is a release implementation which provides version information with TFRegistryProvider Release.
type TFRegistryProviderRelease struct {
	// api is an instance of tfregistry.API interface.
	// It can be replaced for testing.
	api tfregistry.API

	// The user or organization the provider is owned by.
	namespace string

	// The type name of the provider.
	providerType string
}

var _ Release = (*TFRegistryProviderRelease)(nil)

// NewTFRegistryProviderRelease is a factory method which returns an TFRegistryProviderRelease instance.
func NewTFRegistryProviderRelease(source string, config tfregistry.Config) (Release, error) {
	s := strings.SplitN(source, "/", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	client, err := tfregistry.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &TFRegistryProviderRelease{
		api:          client,
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
