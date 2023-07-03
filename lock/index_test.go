package lock

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

// mockProviderDownloaderClient is a mock ProviderDownloaderAPI implementation.
type mockProviderDownloaderClient struct {
	called    int
	responses []*ProviderDownloadResponse
	errs      []error
}

var _ ProviderDownloaderAPI = (*mockProviderDownloaderClient)(nil)

func (c *mockProviderDownloaderClient) ProviderDownload(ctx context.Context, req *ProviderDownloadRequest) (*ProviderDownloadResponse, error) { // nolint revive unused-parameter
	res := c.responses[c.called]
	err := c.errs[c.called]
	c.called++
	return res, err
}

func TestIndexGetOrCreateProviderVersion(t *testing.T) {
	targetPlatforms := []string{"darwin_arm64"}
	allPlatforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64", "windows_amd64"}
	client := &mockProviderDownloaderClient{}
	index := NewIndex(client)

	for _, address := range []string{"minamijoyo/dummy", "minamijoyo/null"} {
		for _, version := range []string{"3.2.1", "3.2.2"} {
			res, err := newMockProviderDownloadResponses(address, version, targetPlatforms, allPlatforms)
			if err != nil {
				t.Fatalf("failed to create mockResponses: err = %s", err)
			}
			// duplicate mocked responses
			mockResponses := []*ProviderDownloadResponse{}
			mockResponses = append(mockResponses, res...)
			mockResponses = append(mockResponses, res...)
			mockNoErrors := make([]error, len(targetPlatforms)*2)
			// reuse the mocked client and set the mocked responses
			client.responses = mockResponses
			client.errs = mockNoErrors
			client.called = 0

			// 1st call
			_, err = index.GetOrCreateProviderVersion(context.Background(), address, version, targetPlatforms)
			if err != nil {
				t.Fatalf("%s@%s: failed to call GetOrCreateProviderVersion: err = %s", address, version, err)
			}
			// expect cache miss
			if client.called != len(targetPlatforms) {
				t.Fatalf("%s@%s: api was called %d times, but expected to be called %d times", address, version, client.called, 1)
			}

			// 2nd call
			_, err = index.GetOrCreateProviderVersion(context.Background(), address, version, targetPlatforms)
			if err != nil {
				t.Fatalf("%s@%s: failed to call GetOrCreateProviderVersion: err = %s", address, version, err)
			}
			// expect cache hit
			if client.called != len(targetPlatforms) {
				t.Fatalf("%s@%s: api was called %d times, but expected to be called %d times", address, version, client.called, 1)
			}
		}
	}
}

func TestProviderIndexGetOrCreateProviderVersion(t *testing.T) {
	allPlatforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64", "windows_amd64"}

	cases := []struct {
		desc      string
		address   string
		version   string
		platforms []string
		want      *ProviderVersion
		ok        bool
	}{
		{
			desc:      "simple",
			address:   "minamijoyo/dummy",
			version:   "3.2.1",
			platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			want: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip": "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":  "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			ok: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := newMockProviderDownloadResponses(tc.address, tc.version, tc.platforms, allPlatforms)
			if err != nil {
				t.Fatalf("failed to create mockResponses: err = %s", err)
			}
			// duplicate mocked responses
			mockResponses := []*ProviderDownloadResponse{}
			mockResponses = append(mockResponses, res...)
			mockResponses = append(mockResponses, res...)
			mockNoErrors := make([]error, len(tc.platforms)*2)
			client := &mockProviderDownloaderClient{
				responses: mockResponses,
				errs:      mockNoErrors,
			}
			pi := newProviderIndex(tc.address, client)

			// 1st call
			got, err := pi.getOrCreateProviderVersion(context.Background(), tc.version, tc.platforms)

			if tc.ok && err != nil {
				t.Fatalf("failed to call getOrCreateProviderVersion: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", spew.Sdump(got))
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(ProviderVersion{})); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
			}

			// expect cache miss
			if client.called != len(tc.platforms) {
				t.Fatalf("api was called %d times, but expected to be called %d times", client.called, len(tc.platforms))
			}

			// 2nd call
			cached, err := pi.getOrCreateProviderVersion(context.Background(), tc.version, tc.platforms)

			if tc.ok && err != nil {
				t.Fatalf("failed to call getOrCreateProviderVersion: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", spew.Sdump(cached))
			}

			if diff := cmp.Diff(cached, tc.want, cmp.AllowUnexported(ProviderVersion{})); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(cached), spew.Sdump(tc.want), diff)
			}

			// expect cache hit
			if client.called != len(tc.platforms) {
				t.Fatalf("api was called %d times, but expected to be called %d times", client.called, len(tc.platforms))
			}
		})
	}
}

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

func TestBuildProviderVersion(t *testing.T) {
	allPlatforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64", "windows_amd64"}

	cases := []struct {
		desc     string
		address  string
		version  string
		platform string
		want     *ProviderVersion
		ok       bool
	}{
		{
			desc:     "simple",
			address:  "minamijoyo/dummy",
			version:  "3.2.1",
			platform: "darwin_arm64",
			want: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			ok: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := newMockProviderDownloadResponse(tc.address, tc.version, tc.platform, allPlatforms)
			if err != nil {
				t.Fatalf("failed to create mockResponse: err = %s", err)
			}

			got, err := buildProviderVersion(tc.address, tc.version, tc.platform, res)

			if tc.ok && err != nil {
				t.Fatalf("failed to call buildProviderVersion: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", spew.Sdump(got))
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(ProviderVersion{})); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
			}
		})
	}
}
