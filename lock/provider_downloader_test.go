package lock

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/minamijoyo/tfupdate/tfregistry"
)

func TestProviderDownloaderClientProviderDownload(t *testing.T) {
	downloadPath := "/terraform-provider-dummy/3.2.1/terraform-provider-dummy_3.2.1_darwin_arm64.zip"
	shaSumsPath := "/terraform-provider-dummy/3.2.1/terraform-provider-dummy_3.2.1_SHA256SUMS"

	// create a zip file in memory.
	zipData, err := newMockZipData("terraform-provider-dummy_v3.2.1_x5", "dummy_3.2.1_darwin_arm64")
	if err != nil {
		t.Fatalf("failed to create a zip file in memory: err = %s", err)
	}

	shaSumsData := []byte(`
5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086  terraform-provider-dummy_3.2.1_darwin_arm64.zip
8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e  terraform-provider-dummy_3.2.1_windows_amd64.zip
c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2  terraform-provider-dummy_3.2.1_linux_amd64.zip
fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2  terraform-provider-dummy_3.2.1_darwin_amd64.zip
`)

	mux, mockServerURL := newMockServer()
	mux.HandleFunc(downloadPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write(zipData)
	})
	mux.HandleFunc(shaSumsPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write(shaSumsData)
	})

	cases := []struct {
		desc   string
		client *mockTFRegistryClient
		want   *ProviderDownloadResponse
		ok     bool
	}{
		{
			desc: "simple",
			client: &mockTFRegistryClient{
				metadataRes: &tfregistry.ProviderPackageMetadataResponse{
					Filename:    "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
					DownloadURL: mockServerURL.String() + downloadPath,
					SHASum:      sha256sumAsHexString(zipData),
					SHASumsURL:  mockServerURL.String() + shaSumsPath,
				},
				err: nil,
			},
			want: &ProviderDownloadResponse{
				filename:    "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
				zipData:     zipData,
				shaSumsData: shaSumsData,
			},
			ok: true,
		},
		{
			desc: "not found",
			client: &mockTFRegistryClient{
				metadataRes: nil,
				err:         errors.New(`unexpected HTTP Status Code: 404`),
			},
			want: nil,
			ok:   false,
		},
		{
			desc: "checksum missmatch",
			client: &mockTFRegistryClient{
				metadataRes: &tfregistry.ProviderPackageMetadataResponse{
					Filename:    "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
					DownloadURL: mockServerURL.String() + downloadPath,
					SHASum:      "aaa",
					SHASumsURL:  mockServerURL.String() + shaSumsPath,
				},
				err: nil,
			},
			want: nil,
			ok:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			config := TFRegistryConfig{
				api: tc.client,
			}
			client := newTestClient(mockServerURL, config)

			req := &ProviderDownloadRequest{
				Namespace: "minamijoyo",
				Type:      "dummy",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			}

			got, err := client.ProviderDownload(context.Background(), req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ProviderDownload: err = %s, req = %s", err, spew.Sdump(req))
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: req = %s, got = %s", spew.Sdump(req), spew.Sdump(got))
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got=%s, but want=%s", spew.Sdump(got), spew.Sdump(tc.want))
			}
		})
	}
}

func TestProviderDownloaderClientDownload(t *testing.T) {
	subPath := "/terraform-provider-dummy/3.2.1/terraform-provider-dummy_3.2.1_darwin_arm64.zip"
	cases := []struct {
		desc    string
		subPath string
		ok      bool
		code    int
		res     []byte
		want    []byte
	}{
		{
			desc:    "simple",
			subPath: subPath,
			ok:      true,
			code:    200,
			// A byte sequence of zip should be returned, but for testing it does not
			// have to be really a zip format, so we use a dummy string here for easy
			// comparison in case of failure.
			res:  []byte("dummy"),
			want: []byte("dummy"),
		},
		{
			desc:    "not found",
			subPath: subPath,
			ok:      false,
			code:    404,
			res:     []byte(""),
			want:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			mux, mockServerURL := newMockServer()
			config := TFRegistryConfig{
				api: &mockTFRegistryClient{},
			}
			client := newTestClient(mockServerURL, config)
			mux.HandleFunc(tc.subPath, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write(tc.res)
			})

			mockServerURL.Path = subPath
			reqURL := mockServerURL.String()
			got, err := client.download(context.Background(), reqURL)

			if tc.ok && err != nil {
				t.Fatalf("failed to call download: err = %s, req = %#v", err, tc.subPath)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: req = %#v, got = %#v", tc.subPath, got)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got=%#v, but want=%#v", got, tc.want)
			}
		})
	}
}

func TestValidateSHA256Sum(t *testing.T) {
	// create a zip file in memory.
	zipData, err := newMockZipData("terraform-provider-dummy_v3.2.1_x5", "dummy_3.2.1_darwin_arm64")
	if err != nil {
		t.Fatalf("failed to create a zip file in memory: err = %s", err)
	}

	cases := []struct {
		desc      string
		b         []byte
		sha256sum string
		ok        bool
	}{
		{
			desc:      "simple",
			b:         zipData,
			sha256sum: "5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
			ok:        true,
		},
		{
			desc:      "checksum missmatch",
			b:         zipData,
			sha256sum: "aaa",
			ok:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateSHA256Sum(tc.b, tc.sha256sum)

			if tc.ok && err != nil {
				t.Fatalf("failed to validate sha256sum: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatal("expected to fail, but success")
			}
		})
	}
}

func TestValidateSHASumsData(t *testing.T) {
	shaSumsData := []byte(`
5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086  terraform-provider-dummy_3.2.1_darwin_arm64.zip
8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e  terraform-provider-dummy_3.2.1_windows_amd64.zip
c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2  terraform-provider-dummy_3.2.1_linux_amd64.zip
fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2  terraform-provider-dummy_3.2.1_darwin_amd64.zip
`)

	cases := []struct {
		desc      string
		b         []byte
		filename  string
		sha256sum string
		ok        bool
	}{
		{
			desc:      "simple",
			b:         shaSumsData,
			filename:  "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
			sha256sum: "5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
			ok:        true,
		},
		{
			desc:      "checksum missmatch",
			b:         shaSumsData,
			filename:  "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
			sha256sum: "aaa",
			ok:        false,
		},
		{
			desc:      "not found",
			b:         shaSumsData,
			filename:  "foo.zip",
			sha256sum: "5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
			ok:        false,
		},
		{
			desc: "parse error",
			b: []byte(`
5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086
`),
			filename:  "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
			sha256sum: "5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
			ok:        false,
		},
		{
			desc:      "empty",
			b:         []byte(""),
			filename:  "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
			sha256sum: "5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
			ok:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateSHASumsData(tc.b, tc.filename, tc.sha256sum)

			if tc.ok && err != nil {
				t.Fatalf("failed to validate SHASumsData: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatal("expected to fail, but success")
			}
		})
	}
}
