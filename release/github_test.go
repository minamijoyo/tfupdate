package release

import (
	"context"
	"errors"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-github/v28/github"
)

// GitHubClient is a mock GitHubAPI implementation.
type mockGitHubClient struct {
	repositoryRelease *github.RepositoryRelease
	response          *github.Response
	err               error
}

func (c *mockGitHubClient) RepositoriesGetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
	return c.repositoryRelease, c.response, c.err
}

func TestNewGitHubRelease(t *testing.T) {
	cases := []struct {
		source string
		owner  string
		repo   string
		ok     bool
	}{
		{
			source: "hoge/fuga",
			owner:  "hoge",
			repo:   "fuga",
			ok:     true,
		},
		{
			source: "hoge",
			owner:  "",
			repo:   "",
			ok:     false,
		},
	}

	client := &mockGitHubClient{}
	for _, tc := range cases {
		got, err := NewGitHubRelease(client, tc.source)

		if tc.ok && err != nil {
			t.Errorf("NewGitHubRelease() with source = %s returns unexpected err: %s", tc.source, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewGitHubRelease() with source = %s expect to return an error, but no error", tc.source)
		}

		if tc.ok {
			r := got.(*GitHubRelease)
			if r.api == nil {
				t.Errorf("NewGitHubRelease() with source = %s expect to set api, but nil", tc.source)
			}

			if !(r.owner == tc.owner && r.repo == tc.repo) {
				t.Errorf("NewGitHubRelease() with source = %s returns (%s, %s), but want (%s, %s)", tc.source, r.owner, r.repo, tc.owner, tc.repo)
			}
		}
	}
}

func TestGitHubReleaseLatest(t *testing.T) {
	tagv010 := "v0.1.0"
	tag010 := "0.1.0"
	cases := []struct {
		client mockGitHubClient
		want   string
		ok     bool
	}{
		{
			client: mockGitHubClient{
				repositoryRelease: &github.RepositoryRelease{
					TagName: &tagv010,
				},
				response: &github.Response{},
				err:      nil,
			},
			want: "0.1.0",
			ok:   true,
		},
		{
			client: mockGitHubClient{
				repositoryRelease: &github.RepositoryRelease{
					TagName: &tag010,
				},
				response: &github.Response{},
				err:      nil,
			},
			want: "0.1.0",
			ok:   true,
		},
		{
			client: mockGitHubClient{
				repositoryRelease: nil,
				response:          &github.Response{},
				// Actual error response type is *github.ErrorResponse,
				// but we are not interested in the internal structure.
				err: errors.New(`GET https://api.github.com/repos/hoge/fuga/releases/latest: 404 Not Found []`),
			},
			want: "",
			ok:   false,
		},
	}

	source := "hoge/fuga"
	for _, tc := range cases {
		r, err := NewGitHubRelease(&tc.client, source)
		if err != nil {
			t.Fatalf("failed to NewGitHubRelease(%#v, %s): %s", tc.client, source, err)
		}

		got, err := r.Latest()

		if tc.ok && err != nil {
			t.Errorf("(*GitHubRelease).Latest() with r = %s returns unexpected err: %+v", spew.Sdump(r), err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*GitHubRelease).Latest() with r = %s expect to return an error, but no error", spew.Sdump(r))
		}

		if got != tc.want {
			t.Errorf("(*GitHubRelease).Latest() with r = %s returns %s, but want = %s", spew.Sdump(r), got, tc.want)
		}
	}
}
