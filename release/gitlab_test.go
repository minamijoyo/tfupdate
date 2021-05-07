package release

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/xanzy/go-gitlab"
)

// mockGitLabClient is a mock GitLabAPI implementation.
type mockGitLabClient struct {
	projectReleases []*gitlab.Release
	response        *gitlab.Response
	err             error
}

// ProjectListReleases returns a list of releases for the mockGitLabClient.
func (c *mockGitLabClient) ProjectListReleases(ctx context.Context, owner, repo string, opt *gitlab.ListReleasesOptions) ([]*gitlab.Release, *gitlab.Response, error) {
	return c.projectReleases, c.response, c.err
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
	tagv := []string{"v0.3.0", "v0.2.0", "v0.1.0"}
	cases := []struct {
		client *mockGitLabClient
		want   string
		ok     bool
	}{ // test v0.1.0 release
		{
			client: &mockGitLabClient{
				projectReleases: []*gitlab.Release{
					&gitlab.Release{TagName: tagv[0]},
					&gitlab.Release{TagName: tagv[1]},
					&gitlab.Release{TagName: tagv[2]},
				},
				response: &gitlab.Response{},
				err:      nil,
			},
			want: "0.3.0",
			ok:   true,
		},
		// test no release
		{
			client: &mockGitLabClient{
				projectReleases: []*gitlab.Release{},
				response:        &gitlab.Response{},
				err:             errors.New("no releases found for project"),
			},
			want: "",
			ok:   false,
		},
		// test unreachable/invalid project
		{
			client: &mockGitLabClient{
				response: &gitlab.Response{},
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

// Test of GitLabRelease.List(ctx context.Context, maxLength int)
func TestGitLabReleaseList(t *testing.T) {
	tagv := []string{"v0.3.0", "v0.2.0", "v0.1.0"}
	cases := []struct {
		client     *mockGitLabClient
		maxLength  int
		preRelease bool
		want       []string
		ok         bool
	}{ // test len(versions) < maxLength
		{
			client: &mockGitLabClient{
				projectReleases: []*gitlab.Release{
					&gitlab.Release{TagName: tagv[0]},
					&gitlab.Release{TagName: tagv[1]},
					&gitlab.Release{TagName: tagv[2]},
				},
				response: &gitlab.Response{},
				err:      nil,
			},
			maxLength:  5,
			preRelease: true,
			want:       []string{"0.1.0", "0.2.0", "0.3.0"}, // reverse order
			ok:         true,
		},
		// test len(versions) > maxLength
		{
			client: &mockGitLabClient{
				projectReleases: []*gitlab.Release{
					&gitlab.Release{TagName: tagv[0]},
					&gitlab.Release{TagName: tagv[1]},
					&gitlab.Release{TagName: tagv[2]},
				},
				response: &gitlab.Response{},
				err:      nil,
			},
			maxLength:  2,
			preRelease: true,
			want:       []string{"0.2.0", "0.3.0"},
			ok:         true,
		},
		// test unreachable/invalid project
		{
			client: &mockGitLabClient{
				projectReleases: nil,
				response:        &gitlab.Response{},
				// Actual error response type is *gitlab.ErrorResponse,
				// but we are not interested in the internal structure.
				err: errors.New(`GET https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab/releases: 404 Not Found []`),
			},
			want: nil,
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

		got, err := r.List(context.Background(), tc.maxLength, tc.preRelease)

		if tc.ok && err != nil {
			t.Errorf("(*GitLabRelease).List() with r = %s, maxLength = %d returns unexpected err: %+v", spew.Sdump(r), tc.maxLength, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*GitLabRelease).List() with r = %s, maxLength = %d expects to return an error, but no error", spew.Sdump(r), tc.maxLength)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("(*GitLabRelease).List() with r = %s, maxLength = %d returns %s, but want = %s", spew.Sdump(r), tc.maxLength, got, tc.want)
		}
	}
}
