package release

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
)

// mockGitHubClient is a mock GitHubAPI implementation.
type mockGitHubClient struct {
	repositoryReleases []*github.RepositoryRelease
	response           *github.Response
	err                error
}

var _ GitHubAPI = (*mockGitHubClient)(nil)

func (c *mockGitHubClient) RepositoriesListReleases(ctx context.Context, owner, repo string, opt *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) { // nolint revive unused-parameter
	return c.repositoryReleases, c.response, c.err
}

func TestNewGitHubClient(t *testing.T) {
	cases := []struct {
		baseURL string
		want    string
		ok      bool
	}{
		{
			baseURL: "",
			want:    "https://api.github.com/",
			ok:      true,
		},
		{
			baseURL: "https://api.github.com/",
			want:    "https://api.github.com/",
			ok:      true,
		},
		{
			baseURL: "http://localhost/",
			want:    "http://localhost/",
			ok:      true,
		},
		{
			baseURL: `https://api\.github.om/`,
			want:    "",
			ok:      false,
		},
	}

	for _, tc := range cases {
		config := GitHubConfig{
			BaseURL: tc.baseURL,
		}
		got, err := NewGitHubClient(config)

		if tc.ok && err != nil {
			t.Errorf("NewGitHubClient() with baseURL = %s returns unexpected err: %s", tc.baseURL, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewGitHubClient() with baseURL = %s expects to return an error, but no error", tc.baseURL)
		}

		if tc.ok {
			if got.client.BaseURL.String() != tc.want {
				t.Errorf("NewGitHubClient() with baseURL = %s returns %s, but want %s", tc.baseURL, got.client.BaseURL.String(), tc.want)
			}
		}
	}
}

func TestNewOAuth2Client(t *testing.T) {
	cases := []struct {
		token string
	}{
		{
			token: "hoge",
		},
	}

	for _, tc := range cases {
		c := newOAuth2Client(tc.token)
		trans := c.Transport.(*oauth2.Transport)
		got, err := trans.Source.Token()
		if err != nil {
			t.Fatalf("failed to get a token from OAuth2 client: %s", err)
		}
		if got.AccessToken != tc.token {
			t.Errorf("newOAuth2Client() expects to set a token = %s, but got = %s", tc.token, got.AccessToken)
		}
	}
}

func TestNewGitHubRelease(t *testing.T) {
	cases := []struct {
		source string
		api    GitHubAPI
		owner  string
		repo   string
		ok     bool
	}{
		{
			source: "hoge/fuga",
			api:    &mockGitHubClient{},
			owner:  "hoge",
			repo:   "fuga",
			ok:     true,
		},
		{
			source: "hoge",
			api:    &mockGitHubClient{},
			owner:  "",
			repo:   "",
			ok:     false,
		},
	}

	for _, tc := range cases {
		config := GitHubConfig{
			api: tc.api,
		}
		got, err := NewGitHubRelease(tc.source, config)

		if tc.ok && err != nil {
			t.Errorf("NewGitHubRelease() with source = %s, api = %#v returns unexpected err: %s", tc.source, tc.api, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewGitHubRelease() with source = %s, api = %#v expects to return an error, but no error", tc.source, tc.api)
		}

		if tc.ok {
			r := got.(*GitHubRelease)

			if r.api != tc.api {
				t.Errorf("NewGitHubRelease() with source = %s, api = %#v sets api = %#v, but want %s", tc.source, tc.api, r.api, tc.api)
			}

			if !(r.owner == tc.owner && r.repo == tc.repo) {
				t.Errorf("NewGitHubRelease() with source = %s, api = %#v returns (%s, %s), but want (%s, %s)", tc.source, tc.api, r.owner, r.repo, tc.owner, tc.repo)
			}
		}
	}
}

func TestGitHubReleaseListReleases(t *testing.T) {
	tagv := []string{"v0.3.0", "v0.2.0", "v0.1.0"}
	cases := []struct {
		client *mockGitHubClient
		want   []string
		ok     bool
	}{
		{
			client: &mockGitHubClient{
				repositoryReleases: []*github.RepositoryRelease{
					{TagName: &tagv[0]},
					{TagName: &tagv[1]},
					{TagName: &tagv[2]},
				},
				response: &github.Response{},
				err:      nil,
			},
			want: []string{"0.3.0", "0.2.0", "0.1.0"},
			ok:   true,
		},
		{
			client: &mockGitHubClient{
				repositoryReleases: nil,
				response:           &github.Response{},
				// Actual error response type is *github.ErrorResponse,
				// but we are not interested in the internal structure.
				err: errors.New(`GET https://api.github.com/repos/hoge/fuga/releases: 404 Not Found []`),
			},
			want: nil,
			ok:   false,
		},
	}

	source := "hoge/fuga"
	for _, tc := range cases {
		// Set a mock client
		config := GitHubConfig{
			api: tc.client,
		}
		r, err := NewGitHubRelease(source, config)
		if err != nil {
			t.Fatalf("failed to NewGitHubRelease(%s, %#v): %s", source, config, err)
		}

		got, err := r.ListReleases(context.Background())

		if tc.ok && err != nil {
			t.Errorf("(*GitHubRelease).ListReleases() with r = %s returns unexpected err: %+v", spew.Sdump(r), err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*GitHubRelease).ListReleases() with r = %s expects to return an error, but no error", spew.Sdump(r))
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("(*GitHubRelease).ListReleases() with r = %s returns %s, but want = %s", spew.Sdump(r), got, tc.want)
		}
	}
}
