package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v28/github"
)

// GitHubRelease is a release implementation which provides version information with GitHub Release.
type GitHubRelease struct {
	client *github.Client
	owner  string
	repo   string
}

// NewGitHubRelease is a factory method which returns an GitHubRelease instance.
func NewGitHubRelease(source string) (Release, error) {
	var owner, repo string

	s := strings.SplitN(source, "/", 2)
	switch len(s) {
	case 2:
		owner = s[0]
		repo = s[1]
	default:
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	return &GitHubRelease{
		client: github.NewClient(nil),
		owner:  owner,
		repo:   repo,
	}, nil
}

// Latest returns a latest version.
func (r *GitHubRelease) Latest() (string, error) {
	release, _, err := r.client.Repositories.GetLatestRelease(context.Background(), r.owner, r.repo)

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
