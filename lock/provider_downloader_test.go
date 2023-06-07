package lock

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

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
			client := newTestClient(mockServerURL)
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
