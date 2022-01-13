package release

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/xanzy/go-gitlab"
)

// GitLabAPI is an interface which calls GitLab API.
// This abstraction layer is needed for testing with mock.
type GitLabAPI interface {
	// ProjectListReleases gets a pagenated of releases accessible by the authenticated user.
	ProjectListReleases(ctx context.Context, owner, project string, opt *gitlab.ListReleasesOptions) ([]*gitlab.Release, *gitlab.Response, error)
}

// GitLabConfig is a set of configurations for GitLabRelease..
type GitLabConfig struct {
	// api is an instance of GitLabAPI interface.
	// It can be replaced for testing.
	api GitLabAPI

	// BaseURL is a URL for GitLab API requests.
	// Defaults to the public GitLab API.
	// BaseURL should always be specified with a trailing slash.
	BaseURL string

	// Token is a personal access token for GitLab, needed to use the api.
	Token string
}

// GitLabClient is a real GitLabAPI implementation.
type GitLabClient struct {
	client *gitlab.Client
}

var _ GitLabAPI = (*GitLabClient)(nil)

// NewGitLabClient returns a real GitLab instance.
func NewGitLabClient(config GitLabConfig) (*GitLabClient, error) {
	if len(config.Token) == 0 {
		return nil, fmt.Errorf("failed to get personal access token (env: GITLAB_TOKEN)")
	}
	c := gitlab.NewClient(nil, config.Token)

	if len(config.BaseURL) != 0 {
		baseURL, err := url.Parse(config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gitlab base url: %s", err)
		}
		if err = c.SetBaseURL(baseURL.String()); err != nil {
			return nil, err
		}
	}

	return &GitLabClient{
		client: c,
	}, nil
}

// ProjectListReleases gets a pagenated of releases accessible by the authenticated user.
func (c *GitLabClient) ProjectListReleases(ctx context.Context, owner, project string, opt *gitlab.ListReleasesOptions) ([]*gitlab.Release, *gitlab.Response, error) {
	return c.client.Releases.ListReleases(owner+"/"+project, opt, gitlab.WithContext(ctx))
}

// GitLabRelease is a release implementation which provides version information with GitLab Release.
type GitLabRelease struct {
	// api is an instance of GitLabAPI interface.
	// It can be replaced for testing.
	api GitLabAPI

	// owner is a namespace of project.
	// limited to one level (group or personal - not sub-groups?)
	owner string

	// project is a name of project (repository).
	project string
}

var _ Release = (*GitLabRelease)(nil)

// NewGitLabRelease is a factory method which returns an GitLabRelease instance.
func NewGitLabRelease(source string, config GitLabConfig) (*GitLabRelease, error) {
	s := strings.SplitN(source, "/", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("failed to parse source: %s", source)
	}

	// If config.api is not set, create a default GitLabClient
	var api GitLabAPI
	if config.api == nil {
		var err error
		api, err = NewGitLabClient(config)
		if err != nil {
			return nil, err
		}
	} else {
		api = config.api
	}

	return &GitLabRelease{
		api:     api,
		owner:   s[0],
		project: s[1],
	}, nil
}

// ListReleases returns a list of unsorted all releases including pre-release.
func (r *GitLabRelease) ListReleases(ctx context.Context) ([]string, error) {
	versions := []string{}
	opt := &gitlab.ListReleasesOptions{
		PerPage: 100, // max
	}

	for {
		releases, resp, err := r.api.ProjectListReleases(ctx, r.owner, r.project, opt)

		if err != nil {
			return nil, fmt.Errorf("failed to list releases for %s/%s: %s", r.owner, r.project, err)
		}

		for _, release := range releases {
			v := tagNameToVersion(release.TagName)
			versions = append(versions, v)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return versions, nil
}
