package release

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
)

// GitHubRelease is a release implementation which provides version information with GitHub Release.
type GitHubRelease struct {
	client *github.Client
	owner  string
	repo   string
}

// NewGitHubRelease is a factory method which returns an GitHubRelease instance.
func NewGitHubRelease(url string) (Release, error) {
	re := regexp.MustCompile(`https://github.com/(.+)/(.+)`)
	matched := re.FindStringSubmatch(url)
	if len(matched) != 3 {
		return nil, errors.Errorf("failed to parse url: %s, matched: %#v", url, matched)
	}
	owner := matched[1]
	repo := matched[2]

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

	name := *release.Name

	// if a name starts with `v`, remove it.
	if name[0] == 'v' {
		return name[1:], nil
	}

	return name, nil
}
