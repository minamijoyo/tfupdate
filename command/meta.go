package command

import (
	"fmt"
	"strings"

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
	case "artifactory":
		s := strings.Split(source, "/")
		if len(s) == 4 {
			config := release.ArtifactoryConfig{
				// Artifactory api format requires us to prefix the TF api with /api/terraform
				BaseURL: fmt.Sprintf("https://%s/artifactory/api/terraform", s[0]),
			}
			return release.NewArtifactoryModuleRelease(source, config)
		}
		return nil, fmt.Errorf("invalid artifactory source - must match ARTIFACTORY_URL/REPOSITORY__NAMESPACE/MODULE_NAME/PROVIDER_NAME - got: %s", source)
	default:
		return nil, fmt.Errorf("failed to new release data source. unknown type: %s", sourceType)
	}
}
