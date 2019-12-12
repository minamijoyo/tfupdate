package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v28/github"
)

// GitHubAPI is an interface which calls GitHub API.
// This abstraction layer is needed for testing with mock.
type GitHubAPI interface {
	// RepositoriesGetLatestRelease fetches the latest published release for the repository.
	// GitHub API docs: https://developer.github.com/v3/repos/releases/#get-the-latest-release
	RepositoriesGetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

// GitHubClient is a real GitHubAPI implementation.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient returns a real GitHubClient instance.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		client: github.NewClient(nil),
	}
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
func NewGitHubRelease(api GitHubAPI, source string) (Release, error) {
	s := strings.SplitN(source, "/", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	return &GitHubRelease{
		api:   api,
		owner: s[0],
		repo:  s[1],
	}, nil
}

// Latest returns a latest version.
func (r *GitHubRelease) Latest() (string, error) {
	release, _, err := r.api.RepositoriesGetLatestRelease(context.Background(), r.owner, r.repo)

	if err != nil {
		return "", fmt.Errorf("failed to get the latest release from github.com/%s/%s: %s", r.owner, r.repo, err)
	}

	// Use TagName because some releases do not have Name.
	tagName := *release.TagName

	// if a tagName starts with `v`, remove it.
	if tagName[0] == 'v' {
		return tagName[1:], nil
	}

	return tagName, nil
}
