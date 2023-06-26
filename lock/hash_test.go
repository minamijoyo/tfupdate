package lock

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

func TestZipDataToH1Hash(t *testing.T) {
	filename := "terraform-provider-dummy_v3.2.1_x5"
	cases := []struct {
		desc     string
		makeZip  bool
		filename string
		// Actually it's a binary of the provider's executable, but here we'll use dummy data for testing.
		contents string
		want     string
		ok       bool
	}{
		{
			desc:     "darwin_arm64",
			makeZip:  true,
			contents: "dummy_3.2.1_darwin_arm64",
			want:     "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
			ok:       true,
		},
		{
			desc:     "darwin_amd64",
			makeZip:  true,
			contents: "dummy_3.2.1_darwin_amd64",
			want:     "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
			ok:       true,
		},
		{
			desc:     "linux_amd64",
			makeZip:  true,
			contents: "dummy_3.2.1_linux_amd64",
			want:     "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
			ok:       true,
		},
		{
			desc:     "invalid zip format",
			makeZip:  false,
			contents: "dummy_3.2.1_linux_amd64",
			want:     "",
			ok:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var zipData []byte
			var err error
			if tc.makeZip {
				// create a zip file in memory.
				zipData, err = newMockZipData(filename, tc.contents)
				if err != nil {
					t.Fatalf("failed to create a zip file in memory: err = %s", err)
				}
			} else {
				// invalid zip format
				zipData = []byte(tc.contents)
			}

			got, err := zipDataToH1Hash(zipData)

			if tc.ok && err != nil {
				t.Fatalf("failed to call zipDataToH1Hash: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", got)
			}

			if got != tc.want {
				t.Errorf("got=%s, but want=%s", got, tc.want)
			}
		})
	}
}

func TestShaSumsDataToZhHash(t *testing.T) {
	// create a valid dummy shaSumsData.
	platforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64", "windows_amd64"}
	shaSumsData, err := newMockShaSumsData("dummy", "3.2.1", platforms)
	if err != nil {
		t.Fatalf("failed to create a shaSumsData: err = %s", err)
	}

	// To update the following static test case, uncomment out here.
	// t.Logf("%s", string(shaSumsData))

	cases := []struct {
		desc        string
		shaSumsData []byte
		want        map[string]string
		ok          bool
	}{
		{
			desc: "static",
			// The input shaSumsData should be the same as the following dynamic
			// case, but the output of newMockShaSumsData is pasted into the test
			// case for test case readability.
			shaSumsData: []byte(`
5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086  terraform-provider-dummy_3.2.1_darwin_arm64.zip
8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e  terraform-provider-dummy_3.2.1_windows_amd64.zip
c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2  terraform-provider-dummy_3.2.1_linux_amd64.zip
fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2  terraform-provider-dummy_3.2.1_darwin_amd64.zip
`),
			want: map[string]string{
				"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
				"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
				"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
				"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
			},
			ok: true,
		},
		{
			desc:        "dynamic",
			shaSumsData: shaSumsData,
			want: map[string]string{
				"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
				"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
				"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
				"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
			},
			ok: true,
		},
		{
			desc:        "empty",
			shaSumsData: []byte(""),
			want:        map[string]string{},
			ok:          true,
		},
		{
			desc:        "parse hash error",
			shaSumsData: []byte("aaa"),
			want:        nil,
			ok:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {

			got, err := shaSumsDataToZhHash(tc.shaSumsData)

			if tc.ok && err != nil {
				t.Fatalf("failed to call shaSumsDataToZhHash: err = %s", err)
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
