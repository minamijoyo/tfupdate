package command

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"github.com/minamijoyo/tfupdate/release"
	"github.com/mitchellh/cli"
	"github.com/spf13/afero"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	// UI is a user interface representing input and output.
	UI cli.Ui

	// Fs is an afero filesystem.
	Fs afero.Fs
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
		config := release.TFRegistryConfig{}
		return release.NewTFRegistryModuleRelease(source, config)
	case "tfregistryProvider":
		config := release.TFRegistryConfig{}
		return release.NewTFRegistryProviderRelease(source, config)
	default:
		return nil, fmt.Errorf("failed to new release data source. unknown type: %s", sourceType)
	}
}
