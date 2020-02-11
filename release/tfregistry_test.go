package release

import (
	"context"
	"errors"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/minamijoyo/tfupdate/tfregistry"
)

// TFRegistryClient is a mock TFRegistryAPI implementation.
type mockTFRegistryClient struct {
	moduleRes *tfregistry.ModuleLatestForProviderResponse
	err       error
}

func (c *mockTFRegistryClient) ModuleLatestForProvider(ctx context.Context, req *tfregistry.ModuleLatestForProviderRequest) (*tfregistry.ModuleLatestForProviderResponse, error) {
	return c.moduleRes, c.err
}

func TestNewTFRegistryClient(t *testing.T) {
	cases := []struct {
		baseURL string
		want    string
		ok      bool
	}{
		{
			baseURL: "",
			want:    "https://registry.terraform.io/",
			ok:      true,
		},
		{
			baseURL: "https://registry.terraform.io/",
			want:    "https://registry.terraform.io/",
			ok:      true,
		},
		{
			baseURL: "http://localhost/",
			want:    "http://localhost/",
			ok:      true,
		},
		{
			baseURL: `https://registry\.terraform.io/`,
			want:    "",
			ok:      false,
		},
	}

	for _, tc := range cases {
		config := TFRegistryConfig{
			BaseURL: tc.baseURL,
		}
		got, err := NewTFRegistryClient(config)

		if tc.ok && err != nil {
			t.Errorf("NewTFRegistryClient() with baseURL = %s returns unexpected err: %s", tc.baseURL, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewTFRegistryClient() with baseURL = %s expects to return an error, but no error", tc.baseURL)
		}

		if tc.ok {
			if got.client.BaseURL.String() != tc.want {
				t.Errorf("NewTFRegistryClient() with baseURL = %s returns %s, but want %s", tc.baseURL, got.client.BaseURL.String(), tc.want)
			}
		}
	}
}

func TestNewTFRegistryModuleRelease(t *testing.T) {
	cases := []struct {
		source    string
		api       TFRegistryAPI
		namespace string
		name      string
		provider  string
		ok        bool
	}{
		{
			source:    "hoge/fuga/piyo",
			api:       &mockTFRegistryClient{},
			namespace: "hoge",
			name:      "fuga",
			provider:  "piyo",
			ok:        true,
		},
		{
			source:    "hoge",
			api:       &mockTFRegistryClient{},
			namespace: "",
			name:      "",
			provider:  "",
			ok:        false,
		},
	}

	for _, tc := range cases {
		config := TFRegistryConfig{
			api: tc.api,
		}
		got, err := NewTFRegistryModuleRelease(tc.source, config)

		if tc.ok && err != nil {
			t.Errorf("NewTFRegistryModuleRelease() with source = %s, api = %#v returns unexpected err: %s", tc.source, tc.api, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewTFRegistryModuleRelease() with source = %s, api = %#v expects to return an error, but no error", tc.source, tc.api)
		}

		if tc.ok {
			r := got.(*TFRegistryModuleRelease)

			if r.api != tc.api {
				t.Errorf("NewTFRegistryModuleRelease() with source = %s, api = %#v sets api = %#v, but want %s", tc.source, tc.api, r.api, tc.api)
			}

			if !(r.namespace == tc.namespace && r.name == tc.name && r.provider == tc.provider) {
				t.Errorf("NewTFRegistryModuleRelease() with source = %s, api = %#v returns (%s, %s, %s), but want (%s, %s, %s)", tc.source, tc.api, r.namespace, r.name, r.provider, tc.namespace, tc.name, tc.provider)
			}
		}
	}
}

func TestTFRegistryModuleReleaseLatest(t *testing.T) {
	cases := []struct {
		client *mockTFRegistryClient
		want   string
		ok     bool
	}{
		{
			client: &mockTFRegistryClient{
				moduleRes: &tfregistry.ModuleLatestForProviderResponse{
					Version: "0.1.0",
				},
				err: nil,
			},
			want: "0.1.0",
			ok:   true,
		},
		{
			client: &mockTFRegistryClient{
				moduleRes: nil,
				err:       errors.New(`{"errors":["Not Found"]}`),
			},
			want: "",
			ok:   false,
		},
	}

	source := "hoge/fuga/piyo"
	for _, tc := range cases {
		// Set a mock client
		config := TFRegistryConfig{
			api: tc.client,
		}
		r, err := NewTFRegistryModuleRelease(source, config)
		if err != nil {
			t.Fatalf("failed to NewTFRegistryModuleRelease(%s, %#v): %s", source, config, err)
		}

		got, err := r.Latest(context.Background())

		if tc.ok && err != nil {
			t.Errorf("(*TFRegistryModuleRelease).Latest() with r = %s returns unexpected err: %+v", spew.Sdump(r), err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*TFRegistryModuleRelease).Latest() with r = %s expects to return an error, but no error", spew.Sdump(r))
		}

		if got != tc.want {
			t.Errorf("(*TFRegistryModuleRelease).Latest() with r = %s returns %s, but want = %s", spew.Sdump(r), got, tc.want)
		}
	}
}
