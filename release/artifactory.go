package release

import (
	"context"
	"fmt"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"net/url"
	"strings"
)

// ArtifactoryAPI is an interface which calls GitLab API.
// This abstraction layer is needed for testing with mock.
type ArtifactoryApi interface {
	ModuleLatestForProvider(ctx context.Context, req *tfregistry.ModuleLatestForProviderRequest) (*tfregistry.ModuleLatestForProviderResponse, error)
}

// GitLabConfig is a set of configurations for GitLabRelease..
type ArtifactoryConfig struct {
	// api is an instance of GitLabAPI interface.
	// It can be replaced for testing.
	api ArtifactoryApi

	// BaseURL is a URL for Artifactory API requests.
	// Defaults to the public Artifactory API.
	// BaseURL should always be specified with a trailing slash.
	BaseURL string

	// Token is a personal access token for Artifactory, needed to use the api.
	Token string
}

// ArtifactoryClient is a real ArtifactoryApi implementation.
type ArtifactoryClient struct {
	client *tfregistry.ArtifactoryClient
}

var _ ArtifactoryApi = (*ArtifactoryClient)(nil)

// NewArtifactoryClient returns a real Artifactory instance.
func NewArtifactoryClient(config ArtifactoryConfig) (*ArtifactoryClient, error) {
	c := tfregistry.NewArtifactoryClient(nil)

	if len(config.BaseURL) != 0 {
		baseURL, err := url.Parse(config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tfregistry base url: %s", err)
		}
		c.BaseURL = baseURL
	}

	return &ArtifactoryClient{
		client: c,
	}, nil
}

// ModuleLatestForProvider gets the latest version of a module for a given provider.
func (c *ArtifactoryClient) ModuleLatestForProvider(ctx context.Context, req *tfregistry.ModuleLatestForProviderRequest) (*tfregistry.ModuleLatestForProviderResponse, error) {
	return c.client.ModuleLatestForProvider(ctx, req)
}

// TFRegistryModuleRelease is a release implementation which provides version information with TFRegistryModule Release.
type ArtifactoryModuleRelease struct {
	// api is an instance of ArtifactoryAPI interface.
	// It can be replaced for testing.
	api ArtifactoryApi

	// namespace is a user name which owns the module.
	namespace string

	// name is a name of the module.
	name string

	// provider is a name of the provider.
	provider string
}

var _ Release = (*ArtifactoryModuleRelease)(nil)

// NewArtifactoryModuleRelease is a factory method which returns an ArtifactoryModuleRelease instance.
func NewArtifactoryModuleRelease(source string, config ArtifactoryConfig) (Release, error) {
	s := strings.Split(source, "/")
	if len(s) != 4 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}
	// If config.api is not set, create a default ArtifactoryClient
	var api ArtifactoryApi
	if config.api == nil {
		var err error
		// Ensure we set the baseURL properly.
		if config.BaseURL == "" {
			config.BaseURL = fmt.Sprintf("https://%s/artifactory/api/terraform", s[0])
		}
		api, err = NewArtifactoryClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		api = config.api
	}

	return &ArtifactoryModuleRelease{
		api:       api,
		namespace: s[1],
		name:      s[2],
		provider:  s[3],
	}, nil
}

// ListReleases returns a list of unsorted all releases including pre-release.
func (r *ArtifactoryModuleRelease) ListReleases(ctx context.Context) ([]string, error) {
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
