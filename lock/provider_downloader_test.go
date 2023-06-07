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
	downloadPath := "/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_darwin_arm64.zip"
	shaSumsPath := "/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_SHA256SUMS"
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
					Filename:    "terraform-provider-null_3.2.1_darwin_arm64.zip",
					DownloadURL: mockServerURL.String() + downloadPath,
					SHASum:      "e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
					SHASumsURL:  mockServerURL.String() + shaSumsPath,
				},
				err: nil,
			},
			want: &ProviderDownloadResponse{
				Data: resData,
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
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			config := TFRegistryConfig{
				api: tc.client,
			}
			client := newTestClient(mockServerURL, config)

			req := &ProviderDownloadRequest{
				Namespace: "hashicorp",
				Type:      "null",
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
