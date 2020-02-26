package tfregistry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestModuleLatestForProvider(t *testing.T) {
	cases := []struct {
		desc string
		req  *ModuleLatestForProviderRequest
		ok   bool
		code int
		res  string
		want *ModuleLatestForProviderResponse
	}{
		{
			desc: "simple",
			req: &ModuleLatestForProviderRequest{
				Namespace: "terraform-aws-modules",
				Name:      "vpc",
				Provider:  "aws",
			},
			ok:   true,
			code: 200,
			res:  `{"version": "2.24.0", "versions": ["2.22.0", "2.23.0", "2.24.0"]}`,
			want: &ModuleLatestForProviderResponse{
				Version:  "2.24.0",
				Versions: []string{"2.22.0", "2.23.0", "2.24.0"},
			},
		},
		{
			desc: "not found",
			req: &ModuleLatestForProviderRequest{
				Namespace: "hoge",
				Name:      "fuga",
				Provider:  "piyo",
			},
			ok:   false,
			code: 404,
			res:  `{"errors":["Not Found"]}`,
			want: nil,
		},
		{
			desc: "invalid request (Namespace)",
			req: &ModuleLatestForProviderRequest{
				Namespace: "",
				Name:      "fuga",
				Provider:  "piyo",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (Name)",
			req: &ModuleLatestForProviderRequest{
				Namespace: "hoge",
				Name:      "",
				Provider:  "piyo",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (Provider)",
			req: &ModuleLatestForProviderRequest{
				Namespace: "hoge",
				Name:      "fuga",
				Provider:  "",
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
			subPath := fmt.Sprintf("%s%s/%s/%s", moduleV1Service, tc.req.Namespace, tc.req.Name, tc.req.Provider)
			mux.HandleFunc(subPath, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.res)
			})

			got, err := client.ModuleLatestForProvider(context.Background(), tc.req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ModuleLatestForProvider: err = %s, req = %#v", err, tc.req)
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
