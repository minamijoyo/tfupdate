package release

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/minamijoyo/tfupdate/tfregistry"
)

// mockTFRegistryClient is a mock TFRegistryAPI implementation.
type mockTFRegistryClient struct {
	moduleRes   *tfregistry.ModuleLatestForProviderResponse
	providerRes *tfregistry.ProviderLatestResponse
	err         error
}

var _ TFRegistryAPI = (*mockTFRegistryClient)(nil)

func (c *mockTFRegistryClient) ModuleLatestForProvider(ctx context.Context, req *tfregistry.ModuleLatestForProviderRequest) (*tfregistry.ModuleLatestForProviderResponse, error) { // nolint revive unused-parameter
	return c.moduleRes, c.err
}

func (c *mockTFRegistryClient) ProviderLatest(ctx context.Context, req *tfregistry.ProviderLatestRequest) (*tfregistry.ProviderLatestResponse, error) { // nolint revive unused-parameter
	return c.providerRes, c.err
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

func TestNewTFRegistryProviderRelease(t *testing.T) {
	cases := []struct {
		source       string
		api          TFRegistryAPI
		namespace    string
		providerType string
		ok           bool
	}{
		{
			source:       "hoge/piyo",
			api:          &mockTFRegistryClient{},
			namespace:    "hoge",
			providerType: "piyo",
			ok:           true,
		},
		{
			source:       "hoge",
			api:          &mockTFRegistryClient{},
			namespace:    "",
			providerType: "",
			ok:           false,
		},
	}

	for _, tc := range cases {
		config := TFRegistryConfig{
			api: tc.api,
		}
		got, err := NewTFRegistryProviderRelease(tc.source, config)

		if tc.ok && err != nil {
			t.Errorf("NewTFRegistryProviderRelease() with source = %s, api = %#v returns unexpected err: %s", tc.source, tc.api, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewTFRegistryProviderRelease() with source = %s, api = %#v expects to return an error, but no error", tc.source, tc.api)
		}

		if tc.ok {
			r := got.(*TFRegistryProviderRelease)

			if r.api != tc.api {
				t.Errorf("NewTFRegistryProviderRelease() with source = %s, api = %#v sets api = %#v, but want %s", tc.source, tc.api, r.api, tc.api)
			}

			if !(r.namespace == tc.namespace && r.providerType == tc.providerType) {
				t.Errorf("NewTFRegistryProviderRelease() with source = %s, api = %#v returns (%s, %s), but want (%s, %s)", tc.source, tc.api, r.namespace, r.providerType, tc.namespace, tc.providerType)
			}
		}
	}
}

func TestTFRegistryModuleReleaseListReleases(t *testing.T) {
	cases := []struct {
		client *mockTFRegistryClient
		want   []string
		ok     bool
	}{
		{
			client: &mockTFRegistryClient{
				moduleRes: &tfregistry.ModuleLatestForProviderResponse{
					Versions: []string{"0.3.0", "0.2.0", "0.1.0"},
				},
				err: nil,
			},
			want: []string{"0.3.0", "0.2.0", "0.1.0"},
			ok:   true,
		},
		{
			client: &mockTFRegistryClient{
				moduleRes: nil,
				err:       errors.New(`unexpected HTTP Status Code: 404`),
			},
			want: nil,
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

		got, err := r.ListReleases(context.Background())

		if tc.ok && err != nil {
			t.Errorf("(*TFRegistryModuleRelease).ListReleases() with r = %s returns unexpected err: %+v", spew.Sdump(r), err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*TFRegistryModuleRelease).ListReleases() with r = %s expects to return an error, but no error", spew.Sdump(r))
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("(*TFRegistryModuleRelease).ListReleases() with r = %s returns %s, but want = %s", spew.Sdump(r), got, tc.want)
		}
	}
}

func TestTFRegistryProviderReleaseListReleases(t *testing.T) {
	cases := []struct {
		client *mockTFRegistryClient
		want   []string
		ok     bool
	}{
		{
			client: &mockTFRegistryClient{
				providerRes: &tfregistry.ProviderLatestResponse{
					Versions: []string{"0.3.0", "0.2.0", "0.1.0"},
				},
				err: nil,
			},
			want: []string{"0.3.0", "0.2.0", "0.1.0"},
			ok:   true,
		},
		{
			client: &mockTFRegistryClient{
				providerRes: nil,
				err:         errors.New(`unexpected HTTP Status Code: 404`),
			},
			want: nil,
			ok:   false,
		},
	}

	source := "hoge/piyo"
	for _, tc := range cases {
		// Set a mock client
		config := TFRegistryConfig{
			api: tc.client,
		}
		r, err := NewTFRegistryProviderRelease(source, config)
		if err != nil {
			t.Fatalf("failed to NewTFRegistryProviderRelease(%s, %#v): %s", source, config, err)
		}

		got, err := r.ListReleases(context.Background())

		if tc.ok && err != nil {
			t.Errorf("(*NewTFRegistryProviderRelease).ListReleases() with r = %s returns unexpected err: %+v", spew.Sdump(r), err)
		}

		if !tc.ok && err == nil {
			t.Errorf("(*NewTFRegistryProviderRelease).ListReleases() with r = %s expects to return an error, but no error", spew.Sdump(r))
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("(*NewTFRegistryProviderRelease).ListReleases() with r = %s returns %s, but want = %s", spew.Sdump(r), got, tc.want)
		}
	}
}
