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
			ok: true,
			res: `{
				"version": "2.24.0"
			}`,
			want: &ModuleLatestForProviderResponse{
				Version: "2.24.0",
			},
		},
	}

	mux, mockServerURL := newMockServer()
	client := newTestClient(mockServerURL)

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			subPath := fmt.Sprintf("/%s/%s/%s/%s", moduleV1Service, tc.req.Namespace, tc.req.Name, tc.req.Provider)
			mux.HandleFunc(subPath, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tc.res)
			})

			got, err := client.ModuleLatestForProvider(context.Background(), tc.req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ModuleLatestForProvider: err = %s, req = %#v", err, tc.req)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: req = %#v", tc.req)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got=%#v, but want=%#v", got, tc.want)
			}
		})
	}
}
