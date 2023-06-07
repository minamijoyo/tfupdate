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
	zipData := []byte("dummy_3.2.1_darwin_arm64")
	shaSumsData := []byte(`
4e064a3094a30c462503e5b589659d46e9b2613d83847dbc5339616b3be26018  terraform-provider-dummy_3.2.1_windows_amd64.zip
6112a5213d3973cb7cdaba1235fdf087f0daa607478fa416fb13766b5d86ab35  terraform-provider-dummy_3.2.1_darwin_amd64.zip
e58101cac36f88a77d20d3192ba7ed81dd3a6af08bd9d7bb52b7568a6b552e4b  terraform-provider-dummy_3.2.1_linux_amd64.zip
d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90  terraform-provider-dummy_3.2.1_darwin_arm64.zip
`)

	mux, mockServerURL := newMockServer()
	mux.HandleFunc(downloadPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(zipData)
	})
	mux.HandleFunc(shaSumsPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(shaSumsData)
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
	subPath := "/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_darwin_arm64.zip"
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
			res:     []byte("dummy"),
			want:    []byte("dummy"),
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
				w.Write(tc.res)
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
	cases := []struct {
		desc      string
		b         []byte
		sha256sum string
		ok        bool
	}{
		{
			desc:      "simple",
			b:         []byte("dummy_3.2.1_darwin_arm64"),
			sha256sum: "d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90",
			ok:        true,
		},
		{
			desc:      "checksum missmatch",
			b:         []byte("dummy_3.2.1_darwin_arm64"),
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
4e064a3094a30c462503e5b589659d46e9b2613d83847dbc5339616b3be26018  terraform-provider-dummy_3.2.1_windows_amd64.zip
6112a5213d3973cb7cdaba1235fdf087f0daa607478fa416fb13766b5d86ab35  terraform-provider-dummy_3.2.1_darwin_amd64.zip
e58101cac36f88a77d20d3192ba7ed81dd3a6af08bd9d7bb52b7568a6b552e4b  terraform-provider-dummy_3.2.1_linux_amd64.zip
d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90  terraform-provider-dummy_3.2.1_darwin_arm64.zip
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
			sha256sum: "d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90",
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
			sha256sum: "d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90",
			ok:        false,
		},
		{
			desc: "parse error",
			b: []byte(`
d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90
`),
			filename:  "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
			sha256sum: "d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90",
			ok:        false,
		},
		{
			desc:      "empty",
			b:         []byte(""),
			filename:  "terraform-provider-dummy_3.2.1_darwin_arm64.zip",
			sha256sum: "d95ca113388ef9530b5f664eb086f798f8eae75047bd0a0eaef00f980fd34c90",
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
