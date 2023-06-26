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
	platform := "darwin_arm64"
	res, err := newMockProviderDownloadResponse(platform)
	if err != nil {
		t.Fatalf("failed to create mockResponses: err = %s", err)
	}
	// duplicate mocked responses
	mockResponses := []*ProviderDownloadResponse{res, res}
	mockNoErrors := []error{nil, nil}

	client := &mockProviderDownloaderClient{
		responses: mockResponses,
		errs:      mockNoErrors,
	}
	index := NewIndex(client)

	for _, address := range []string{"minamijoyo/dummy", "minamijoyo/null"} {
		for _, version := range []string{"3.2.1", "3.2.2"} {
			// In this test case, only the number of calls is verified for testing
			// the cache behavior. A returned valued is intentionally ignored because
			// the address and version are hard-coded in the the mock response.

			// 1st call
			_, err = index.GetOrCreateProviderVersion(context.Background(), address, version, []string{platform})
			if err != nil {
				t.Fatalf("%s@%s: failed to call GetOrCreateProviderVersion: err = %s", address, version, err)
			}
			// expect cache miss
			if client.called != 1 {
				t.Fatalf("%s@%s: api was called %d times, but expected to be called %d times", address, version, client.called, 1)
			}

			// 2nd call
			_, err = index.GetOrCreateProviderVersion(context.Background(), address, version, []string{platform})
			if err != nil {
				t.Fatalf("%s@%s: failed to call GetOrCreateProviderVersion: err = %s", address, version, err)
			}
			// expect cache hit
			if client.called != 1 {
				t.Fatalf("%s@%s: api was called %d times, but expected to be called %d times", address, version, client.called, 1)
			}

			// reset the called counter
			client.called = 0
		}
	}
}

func TestProviderIndexGetOrCreateProviderVersion(t *testing.T) {
	address := "minamijoyo/dummy"
	platforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64"}
	mockResponses, err := newMockProviderDownloadResponses(platforms)
	if err != nil {
		t.Fatalf("failed to create mockResponses: err = %s", err)
	}
	mockNoErrors := []error{nil, nil, nil}

	cases := []struct {
		desc      string
		client    *mockProviderDownloaderClient
		version   string
		platforms []string
		want      *ProviderVersion
		ok        bool
	}{
		{
			desc: "simple",
			client: &mockProviderDownloaderClient{
				responses: mockResponses,
				errs:      mockNoErrors,
			},
			version:   "3.2.1",
			platforms: platforms,
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
			pi := newProviderIndex(address, tc.client)
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
			if tc.client.called != len(tc.platforms) {
				t.Fatalf("api was called %d times, but expected to be called %d times", tc.client.called, len(tc.platforms))
			}

			// 2nd call
			cached, err := pi.getOrCreateProviderVersion(context.Background(), tc.version, tc.platforms)

			if tc.ok && err != nil {
				t.Fatalf("failed to call getOrCreateProviderVersion: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", spew.Sdump(cached))
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(ProviderVersion{})); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(cached), spew.Sdump(tc.want), diff)
			}

			// expect cache hit
			if tc.client.called != len(tc.platforms) {
				t.Fatalf("api was called %d times, but expected to be called %d times", tc.client.called, len(tc.platforms))
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
	mockRes, err := newMockProviderDownloadResponse("darwin_arm64")
	if err != nil {
		t.Fatalf("failed to create mockResponse: err = %s", err)
	}

	cases := []struct {
		desc     string
		address  string
		version  string
		platform string
		res      *ProviderDownloadResponse
		want     *ProviderVersion
		ok       bool
	}{
		{
			desc:     "simple",
			address:  "minamijoyo/dummy",
			version:  "3.2.1",
			platform: "darwin_arm64",
			res:      mockRes,
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
			got, err := buildProviderVersion(tc.address, tc.version, tc.platform, tc.res)

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
