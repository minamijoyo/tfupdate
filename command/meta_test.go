package command

import (
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/minamijoyo/tfupdate/release"
)

func TestNewRelease(t *testing.T) {
	cases := []struct {
		sourceType string
		source     string
		env        map[string]string
		want       release.Release
		ok         bool
	}{
		{
			sourceType: "github",
			source:     "hashicorp/terraform",
			env:        map[string]string{},
			want:       &release.GitHubRelease{},
			ok:         true,
		},
		{
			sourceType: "gitlab",
			source:     "gitlab-org/gitlab",
			env:        map[string]string{"GITLAB_TOKEN": "dummy-token-for-testing"},
			want:       &release.GitLabRelease{},
			ok:         true,
		},
		{
			sourceType: "tfregistryModule",
			source:     "terraform-aws-modules/vpc/aws",
			env:        map[string]string{},
			want:       &release.TFRegistryModuleRelease{},
			ok:         true,
		},
		{
			sourceType: "tfregistryProvider",
			source:     "hashicorp/aws",
			env:        map[string]string{},
			want:       &release.TFRegistryProviderRelease{},
			ok:         true,
		},
		{
			sourceType: "invalid",
			source:     "test",
			env:        map[string]string{},
			want:       nil,
			ok:         false,
		},
		{
			sourceType: "",
			source:     "test",
			env:        map[string]string{},
			want:       nil,
			ok:         false,
		},
	}

	for _, tc := range cases {
		// Set environment variables for this test case
		originalEnv := make(map[string]string)
		for key, value := range tc.env {
			originalEnv[key] = os.Getenv(key)
			os.Setenv(key, value)
		}
		// Clean up environment variables after test
		defer func(env map[string]string, original map[string]string) {
			for key := range env {
				if originalValue, existed := original[key]; existed {
					os.Setenv(key, originalValue)
				} else {
					os.Unsetenv(key)
				}
			}
		}(tc.env, originalEnv)

		got, err := newRelease(tc.sourceType, tc.source)
		if tc.ok && err != nil {
			t.Errorf("newRelease() with sourceType = %s, source = %s returns unexpected err: %+v", tc.sourceType, tc.source, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("newRelease() with sourceType = %s, source = %s expects to return an error, but no error", tc.sourceType, tc.source)
		}

		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(release.GitHubRelease{}),
			cmpopts.IgnoreUnexported(release.GitLabRelease{}),
			cmpopts.IgnoreUnexported(release.TFRegistryModuleRelease{}),
			cmpopts.IgnoreUnexported(release.TFRegistryProviderRelease{}),
		}
		if diff := cmp.Diff(got, tc.want, opts...); diff != "" {
			t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
		}
	}
}
