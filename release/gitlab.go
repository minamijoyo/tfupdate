package release

import (
	"fmt"

	gitlab "github.com/xanzy/go-gitlab"
)

// GitLabRelease is a release implementation which provides version information with GitLab Release.
type GitLabRelease struct {
	client  *gitlab.Client
	owner   string
	project string
}

// NewGitLabRelease is a factory method which returns an GitLabRelease instance.
func NewGitLabRelease(owner string, project string, token string) (Release, error) {
	return &GitLabRelease{
		client:  gitlab.NewClient(nil, token),
		owner:   owner,
		project: project,
	}, nil
}

// Latest returns a latest version.
func (r *GitLabRelease) Latest() (string, error) {
  opt := &gitlab.ListReleasesOptions{}
  releases, _, err := r.client.Releases.ListReleases(1, opt)

	if err != nil {
		return "", fmt.Errorf("failed to get the releases from %s/%s: %s", r.owner, r.project, err)
	}

  // Get latest release
  latest := releases[0]

	// Use TagName because some releases do not have Name.
	tagName := latest.TagName

	// if a tagName starts with `v`, remove it.
	if tagName[0] == 'v' {
		return tagName[1:], nil
	}

	return tagName, nil
}
