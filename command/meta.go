package command

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"github.com/minamijoyo/tfupdate/release"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"github.com/mitchellh/cli"
	"github.com/spf13/afero"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	// UI is a user interface representing input and output.
	UI cli.Ui

	// Fs is an afero filesystem.
	Fs afero.Fs

	// releaseFactory is a factory function to create Release instances.
	// It can be overridden for testing to avoid external API calls.
	// If nil, the default newRelease function will be used.
	releaseFactory func(sourceType, source string) (release.Release, error)
}

// newRelease is a factory method which returns an Release implementation.
func newRelease(sourceType string, source string) (release.Release, error) {
	var env Env
	err := envconfig.Process("", &env)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch environment variables: %s", err)
	}

	switch sourceType {
	case "github":
		config := release.GitHubConfig{
			BaseURL: env.GitHubBaseURL,
			Token:   env.GitHubToken,
		}
		return release.NewGitHubRelease(source, config)
	case "gitlab":
		config := release.GitLabConfig{
			BaseURL: env.GitLabBaseURL,
			Token:   env.GitLabToken,
		}
		return release.NewGitLabRelease(source, config)
	case "tfregistryModule":
		config := tfregistry.Config{
			BaseURL: env.TFRegistryBaseURL,
		}
		return release.NewTFRegistryModuleRelease(source, config)
	case "tfregistryProvider":
		config := tfregistry.Config{
			BaseURL: env.TFRegistryBaseURL,
		}
		return release.NewTFRegistryProviderRelease(source, config)
	default:
		return nil, fmt.Errorf("failed to new release data source. unknown type: %s", sourceType)
	}
}

// NewRelease creates a Release instance using the configured factory or default implementation.
func (m *Meta) NewRelease(sourceType, source string) (release.Release, error) {
	factory := m.releaseFactory
	if factory == nil {
		factory = newRelease
	}
	return factory(sourceType, source)
}
