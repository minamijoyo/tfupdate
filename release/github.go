package release

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

// GitHubAPI is an interface which calls GitHub API.
// This abstraction layer is needed for testing with mock.
type GitHubAPI interface {
	// RepositoriesGetLatestRelease fetches the latest published release for the repository.
	// GitHub API docs: https://developer.github.com/v3/repos/releases/#get-the-latest-release
	RepositoriesGetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

// GitHubConfig is a set of configurations for GitHubRelease.
type GitHubConfig struct {
	// api is an instance of GitHubAPI interface.
	// It can be replaced for testing.
	api GitHubAPI

	// BaseURL is a URL for GitHub API requests.
	// Defaults to the public GitHub API.
	// This looks like the GitHub Enterprise support, but currently for testing purposes only.
	// The GitHub Enterprise is not supported yet.
	// BaseURL should always be specified with a trailing slash.
	BaseURL string

	// Token is a personal access token for GitHub.
	// This allows access to a private repository.
	Token string
}

// GitHubClient is a real GitHubAPI implementation.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient returns a real GitHubClient instance.
func NewGitHubClient(config GitHubConfig) (*GitHubClient, error) {
	var hc *http.Client
	if len(config.Token) != 0 {
		hc = newOAuth2Client(config.Token)
	}
	c := github.NewClient(hc)

	if len(config.BaseURL) != 0 {
		baseURL, err := url.Parse(config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse github base url: %s", err)
		}
		c.BaseURL = baseURL
	}

	return &GitHubClient{
		client: c,
	}, nil
}

// newOAuth2Client returns a *http.Client which sets a given token to the Authorization header.
// This allows access to a private repository.
func newOAuth2Client(token string) *http.Client {
	t := &oauth2.Token{
		AccessToken: token,
	}
	ts := oauth2.StaticTokenSource(t)

	return oauth2.NewClient(context.Background(), ts)
}

// RepositoriesGetLatestRelease fetches the latest published release for the repository.
func (c *GitHubClient) RepositoriesGetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
	return c.client.Repositories.GetLatestRelease(ctx, owner, repo)
}

// GitHubRelease is a release implementation which provides version information with GitHub Release.
type GitHubRelease struct {
	// api is an instance of GitHubAPI interface.
	// It can be replaced for testing.
	api GitHubAPI

	// owner is a namespace of repository.
	owner string

	// repo is a name of repository.
	repo string
}

// NewGitHubRelease is a factory method which returns an GitHubRelease instance.
func NewGitHubRelease(source string, config GitHubConfig) (Release, error) {
	s := strings.SplitN(source, "/", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	// If config.api is not set, create a default GitHubClient
	var api GitHubAPI
	if config.api == nil {
		var err error
		api, err = NewGitHubClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		api = config.api
	}

	return &GitHubRelease{
		api:   api,
		owner: s[0],
		repo:  s[1],
	}, nil
}

// Latest returns a latest version.
func (r *GitHubRelease) Latest(ctx context.Context) (string, error) {
	release, _, err := r.api.RepositoriesGetLatestRelease(ctx, r.owner, r.repo)

	if err != nil {
		return "", fmt.Errorf("failed to get the latest release for %s/%s: %s", r.owner, r.repo, err)
	}

	// Use TagName because some releases do not have Name.
	tagName := *release.TagName

	// if a tagName starts with `v`, remove it.
	if tagName[0] == 'v' {
		return tagName[1:], nil
	}

	return tagName, nil
}
