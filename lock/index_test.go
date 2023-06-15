package lock

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

func TestNewProviderDownloadRequest(t *testing.T) {
	cases := []struct {
		desc     string
		address  string
		version  string
		platform string
		want     *ProviderDownloadRequest
		ok       bool
	}{
		{
			desc:     "simple",
			address:  "minamijoyo/dummy",
			version:  "3.2.1",
			platform: "darwin_arm64",
			want: &ProviderDownloadRequest{
				Namespace: "minamijoyo",
				Type:      "dummy",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok: true,
		},
		{
			desc:     "fully qualified provider address",
			address:  "registry.terraform.io/minamijoyo/dummy",
			version:  "3.2.1",
			platform: "darwin_arm64",
			want: &ProviderDownloadRequest{
				Namespace: "minamijoyo",
				Type:      "dummy",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok: true,
		},
		{
			desc:     "unknown provider namespace",
			address:  "null",
			version:  "3.2.1",
			platform: "darwin_arm64",
			want:     nil,
			ok:       false,
		},
		{
			desc:     "legacy provider namespace",
			address:  "-/null",
			version:  "3.2.1",
			platform: "darwin_arm64",
			want:     nil,
			ok:       false,
		},
		{
			desc:     "zero provider namespace",
			address:  "",
			version:  "3.2.1",
			platform: "darwin_arm64",
			want:     nil,
			ok:       false,
		},
		{
			desc:     "invalid platform",
			address:  "minamijoyo/dummy",
			version:  "3.2.1",
			platform: "foo",
			want:     nil,
			ok:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := newProviderDownloadRequest(tc.address, tc.version, tc.platform)

			if tc.ok && err != nil {
				t.Fatalf("failed to call newProviderDownloadRequest: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", spew.Sdump(got))
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
			}
		})
	}
}
