package tfregistry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestListModuleVersions(t *testing.T) {
	cases := []struct {
		desc string
		req  *ListModuleVersionsRequest
		ok   bool
		code int
		res  string
		want *ListModuleVersionsResponse
	}{
		{
			desc: "simple",
			req: &ListModuleVersionsRequest{
				Namespace: "terraform-aws-modules",
				Name:      "vpc",
				Provider:  "aws",
			},
			ok:   true,
			code: 200,
			res:  `{"modules": [{"versions": [{"version": "2.22.0"}, {"version": "2.23.0"}, {"version": "2.24.0"}]}]}`,
			want: &ListModuleVersionsResponse{
				Modules: []ModuleVersions{
					{
						Versions: []ModuleVersion{
							{Version: "2.22.0"},
							{Version: "2.23.0"},
							{Version: "2.24.0"},
						},
					},
				},
			},
		},
		{
			desc: "not found",
			req: &ListModuleVersionsRequest{
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
			req: &ListModuleVersionsRequest{
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
			req: &ListModuleVersionsRequest{
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
			req: &ListModuleVersionsRequest{
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
			subPath := fmt.Sprintf("%s%s/%s/%s/versions", moduleV1Service, tc.req.Namespace, tc.req.Name, tc.req.Provider)
			mux.HandleFunc(subPath, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.res)
			})

			got, err := client.ListModuleVersions(context.Background(), tc.req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ListModuleVersions: err = %s, req = %#v", err, tc.req)
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
