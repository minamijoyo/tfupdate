package tfregistry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type providerVersionsRoundTripFunc func(*http.Request) (*http.Response, error)

func (f providerVersionsRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type providerVersionsBody struct {
	io.Reader
	closed *bool
}

func (b providerVersionsBody) Close() error {
	*b.closed = true
	return nil
}

func TestListProviderVersions(t *testing.T) {
	cases := []struct {
		desc string
		req  *ListProviderVersionsRequest
		ok   bool
		code int
		res  string
		want *ListProviderVersionsResponse
	}{
		{
			desc: "simple",
			req: &ListProviderVersionsRequest{
				Namespace: "hashicorp",
				Type:      "aws",
			},
			ok:   true,
			code: 200,
			res:  `{"versions": [{"version": "3.5.0"}, {"version": "3.6.0"}, {"version": "3.7.0"}]}`,
			want: &ListProviderVersionsResponse{
				Versions: []ProviderVersion{
					{Version: "3.5.0"},
					{Version: "3.6.0"},
					{Version: "3.7.0"},
				},
			},
		},
		{
			desc: "with protocols and platforms",
			req: &ListProviderVersionsRequest{
				Namespace: "hashicorp",
				Type:      "aws",
			},
			ok:   true,
			code: 200,
			res:  `{"versions": [{"version": "3.7.0", "protocols": ["4.0", "5.1"], "platforms": [{"os": "linux", "arch": "amd64"}, {"os": "darwin", "arch": "amd64"}]}]}`,
			want: &ListProviderVersionsResponse{
				Versions: []ProviderVersion{
					{
						Version:   "3.7.0",
						Protocols: []string{"4.0", "5.1"},
						Platforms: []ProviderPlatform{
							{OS: "linux", Arch: "amd64"},
							{OS: "darwin", Arch: "amd64"},
						},
					},
				},
			},
		},
		{
			desc: "not found",
			req: &ListProviderVersionsRequest{
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
			req: &ListProviderVersionsRequest{
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
			req: &ListProviderVersionsRequest{
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
			subPath := fmt.Sprintf("%s%s/%s/versions", providerV1Service, tc.req.Namespace, tc.req.Type)
			mux.HandleFunc(subPath, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.res)
			})

			got, err := client.ListProviderVersions(context.Background(), tc.req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ListProviderVersions: err = %s, req = %#v", err, tc.req)
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

func TestConstraintLockListProviderVersionsClosesErrorBody(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{name: "success", status: http.StatusOK, body: `{"versions":[{"version":"1.0.0"}]}`},
		{name: "malformed JSON", status: http.StatusOK, body: `{"versions":`, wantErr: true},
		{name: "non-success status", status: http.StatusServiceUnavailable, body: "unavailable", wantErr: true},
	}
	for _, tc := range cases {
		closed := false
		httpClient := &http.Client{Transport: providerVersionsRoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: tc.status,
				Body:       providerVersionsBody{Reader: strings.NewReader(tc.body), closed: &closed},
				Header:     make(http.Header),
			}, nil
		})}
		client, err := NewClient(Config{BaseURL: "https://registry.example/", HTTPClient: httpClient})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.ListProviderVersions(context.Background(), &ListProviderVersionsRequest{Namespace: "acme", Type: "widget"})
		if (err != nil) != tc.wantErr {
			t.Fatalf("%s: error = %v, want error %t", tc.name, err, tc.wantErr)
		}
		if !closed {
			t.Errorf("%s response body was not closed", tc.name)
		}
	}
}
