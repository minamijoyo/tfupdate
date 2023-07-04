package tfregistry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestProviderLatest(t *testing.T) {
	cases := []struct {
		desc string
		req  *ProviderLatestRequest
		ok   bool
		code int
		res  string
		want *ProviderLatestResponse
	}{
		{
			desc: "simple",
			req: &ProviderLatestRequest{
				Namespace: "hashicorp",
				Type:      "aws",
			},
			ok:   true,
			code: 200,
			res:  `{"version": "3.7.0", "versions": ["3.5.0", "3.6.0", "3.7.0"]}`,
			want: &ProviderLatestResponse{
				Version:  "3.7.0",
				Versions: []string{"3.5.0", "3.6.0", "3.7.0"},
			},
		},
		{
			desc: "not found",
			req: &ProviderLatestRequest{
				Namespace: "hoge",
				Type:      "piyo",
			},
			ok:   false,
			code: 404,
			res:  `{"errors":["Not Found"]}`,
			want: nil,
		},
		{
			desc: "invalid request (Namespace)",
			req: &ProviderLatestRequest{
				Namespace: "",
				Type:      "piyo",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (Type)",
			req: &ProviderLatestRequest{
				Namespace: "hoge",
				Type:      "",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			mux, mockServerURL := newMockServer()
			client := newTestClient(mockServerURL)
			subPath := fmt.Sprintf("%s%s/%s", providerV1Service, tc.req.Namespace, tc.req.Type)
			mux.HandleFunc(subPath, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.res)
			})

			got, err := client.ProviderLatest(context.Background(), tc.req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ProviderLatest: err = %s, req = %#v", err, tc.req)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: req = %#v, got = %#v", tc.req, got)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got=%#v, but want=%#v", got, tc.want)
			}
		})
	}
}
