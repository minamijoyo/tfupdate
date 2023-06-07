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
	resData := []byte("dummy")
	mux, mockServerURL := newMockServer()
	mux.HandleFunc(downloadPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(resData)
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
					SHASum:      sha256sumAsHexString(resData),
					SHASumsURL:  mockServerURL.String() + shaSumsPath,
				},
				err: nil,
			},
			want: &ProviderDownloadResponse{
				zipData:   resData,
				SHA256Sum: sha256sumAsHexString(resData),
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
			b:         []byte("dummy"),
			sha256sum: "b5a2c96250612366ea272ffac6d9744aaf4b45aacd96aa7cfcb931ee3b558259",
			ok:        true,
		},
		{
			desc:      "checksum missmatch",
			b:         []byte("dummy"),
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
