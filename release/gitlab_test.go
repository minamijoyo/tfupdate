package release

import (
	"context"
	"errors"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/xanzy/go-gitlab"
)

// mockGitLabClient is a mock GitLabAPI implementation.
type mockGitLabClient struct {
	projectRelease *gitlab.Release
	response       *gitlab.Response
	err            error
}

// ProjectGetLatestRelease returns the latest release for the mockGitLabClient.
func (c *mockGitLabClient) ProjectGetLatestRelease(ctx context.Context, owner, project string) (*gitlab.Release, *gitlab.Response, error) {
	return c.projectRelease, c.response, c.err
}

// Test of NewGitLabClient(config GitLabConfig)
func TestNewGitLabClient(t *testing.T) {
	cases := []struct {
		baseURL string
		want    string
		ok      bool
	}{ // test default value
		{
			baseURL: "",
			want:    "https://gitlab.com/api/v4/",
			ok:      true,
		},
		// test custom value
		{
			baseURL: "https://gitlab.com/api/v4/",
			want:    "https://gitlab.com/api/v4/",
			ok:      true,
		},
		// test custom value
		{
			baseURL: "http://localhost/api/v4/",
			want:    "http://localhost/api/v4/",
			ok:      true,
		},
		// test unparsable URL
		{
			baseURL: `https://gitlab\.com/api/v4/`,
			want:    "",
			ok:      false,
		},
	}

	for _, tc := range cases {
		config := GitLabConfig{
			BaseURL: tc.baseURL,
			Token:   "dummy_token",
		}
		got, err := NewGitLabClient(config)

		if tc.ok && err != nil {
			t.Errorf("NewGitLabClient() with baseURL = %s returns unexpected err: %s", tc.baseURL, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewGitLabClient() with baseURL = %s expects to return an error, but no error", tc.baseURL)
		}

		if tc.ok {
			if got.client.BaseURL().String() != tc.want {
				t.Errorf("NewGitLabClient() with baseURL = %s returns %s, but want %s", tc.baseURL, got.client.BaseURL().String(), tc.want)
			}
		}
	}
}

// Test of NewGitLabRelease(source string, config GitLabConfig)
func TestNewGitLabRelease(t *testing.T) {
	cases := []struct {
		source  string
		api     GitLabAPI
		owner   string
		project string
		ok      bool
	}{ // test complete config
		{
			source:  "gitlab-org/gitlab",
			api:     &mockGitLabClient{},
			owner:   "gitlab-org",
			project: "gitlab",
			ok:      true,
		},
		// test release without owner or project
		{
			source:  "gitlab",
			api:     &mockGitLabClient{},
			owner:   "",
			project: "",
			ok:      false,
		},
		// test release with missing api
		{
			source:  "gitlab-org/gitlab",
			api:     nil,
			owner:   "gitlab-org",
			project: "gitlab",
			ok:      false,
		},
	}

	for _, tc := range cases {
		config := GitLabConfig{
			api: tc.api,
		}
		got, err := NewGitLabRelease(tc.source, config)

		if tc.ok && err != nil {
			t.Errorf("NewGitLabRelease() with source = %s, api = %#v returns unexpected err: %s", tc.source, tc.api, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewGitLabRelease() with source = %s, api = %#v expects to return an error, but no error", tc.source, tc.api)
		}

		if tc.ok {
			r := got

			if r.api != tc.api {
				t.Errorf("NewGitLabRelease() with source = %s, api = %#v sets api = %#v, but want %s", tc.source, tc.api, r.api, tc.api)
			}

			if !(r.owner == tc.owner && r.project == tc.project) {
				t.Errorf("NewGitLabRelease() with source = %s, api = %#v returns (%s, %s), but want (%s, %s)", tc.source, tc.api, r.owner, r.project, tc.owner, tc.project)
			}
		}
	}
}

// Test of GitLabRelease.Latest(ctx context.Context)
func TestGitLabReleaseLatest(t *testing.T) {
	tagv010 := "v0.1.0"
	tag010 := "0.1.0"
	cases := []struct {
		client *mockGitLabClient
		want   string
		ok     bool
	}{ // test v0.1.0 release
		{
			client: &mockGitLabClient{
				projectRelease: &gitlab.Release{
					TagName: tagv010,
				},
				response: &gitlab.Response{},
				err:      nil,
			},
			want: "0.1.0",
			ok:   true,
		},
		// test 0.1.0 release
		{
			client: &mockGitLabClient{
				projectRelease: &gitlab.Release{
					TagName: tag010,
				},
				response: &gitlab.Response{},
				err:      nil,
			},
			want: "0.1.0",
			ok:   true,
		},
		// test no release
		{
			client: &mockGitLabClient{
				projectRelease: &gitlab.Release{},
				response:       &gitlab.Response{},
				err:            errors.New("no releases found for project"),
			},
			want: "",
			ok:   false,
		},
		// test unreachable/invalid project
		{
			client: &mockGitLabClient{
				projectRelease: nil,
				response:       &gitlab.Response{},
				// Actual error response type is *gitlab.ErrorResponse,
				// but we are not interested in the internal structure.
				err: errors.New(`GET https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab/releases: 404 Not Found []`),
			},
			want: "",
			ok:   false,
		},
	}

	source := "gitlab-org/gitlab"
	for _, tc := range cases {
		// Set a mock client
		config := GitLabConfig{
			api: tc.client,
		}
		r, err := NewGitLabRelease(source, config)
		if err != nil {
			t.Fatalf("failed to NewGitLabRelease(%s, %#v): %s", source, config, err)
		}

		got, err := r.Latest(context.Background())

		if tc.ok && err != nil {
			t.Errorf("(*GitLabRelease).Latest() with r = %s returns unexpected err: %+v", spew.Sdump(r), err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*GitLabRelease).Latest() with r = %s expects to return an error, but no error", spew.Sdump(r))
		}

		if got != tc.want {
			t.Errorf("(*GitLabRelease).Latest() with r = %s returns %s, but want = %s", spew.Sdump(r), got, tc.want)
		}
	}
}
