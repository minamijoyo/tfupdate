package release

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/minamijoyo/tfupdate/tfregistry"
)

// mockTFRegistryClient is a mock implementation of tfregistry.API
type mockTFRegistryClient struct {
	moduleRes   *tfregistry.ListModuleVersionsResponse
	providerRes *tfregistry.ListProviderVersionsResponse
	err         error
}

var _ tfregistry.API = (*mockTFRegistryClient)(nil)

func (c *mockTFRegistryClient) ListModuleVersions(_ context.Context, _ *tfregistry.ListModuleVersionsRequest) (*tfregistry.ListModuleVersionsResponse, error) {
	return c.moduleRes, c.err
}

func (c *mockTFRegistryClient) ListProviderVersions(_ context.Context, _ *tfregistry.ListProviderVersionsRequest) (*tfregistry.ListProviderVersionsResponse, error) {
	return c.providerRes, c.err
}

func (c *mockTFRegistryClient) ProviderPackageMetadata(_ context.Context, _ *tfregistry.ProviderPackageMetadataRequest) (*tfregistry.ProviderPackageMetadataResponse, error) {
	return nil, nil // dummy implementation as it's not used in tests
}

func TestNewTFRegistryModuleRelease(t *testing.T) {
	cases := []struct {
		source    string
		namespace string
		name      string
		provider  string
		ok        bool
	}{
		{
			source:    "hoge/fuga/piyo",
			namespace: "hoge",
			name:      "fuga",
			provider:  "piyo",
			ok:        true,
		},
		{
			source:    "hoge",
			namespace: "",
			name:      "",
			provider:  "",
			ok:        false,
		},
	}

	for _, tc := range cases {
		config := tfregistry.Config{}
		got, err := NewTFRegistryModuleRelease(tc.source, config)

		if tc.ok && err != nil {
			t.Errorf("NewTFRegistryModuleRelease() with source = %s returns unexpected err: %s", tc.source, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewTFRegistryModuleRelease() with source = %s expects to return an error, but no error", tc.source)
		}

		if tc.ok {
			r := got.(*TFRegistryModuleRelease)

			if !(r.namespace == tc.namespace && r.name == tc.name && r.provider == tc.provider) {
				t.Errorf("NewTFRegistryModuleRelease() with source = %s returns (%s, %s, %s), but want (%s, %s, %s)", tc.source, r.namespace, r.name, r.provider, tc.namespace, tc.name, tc.provider)
			}
		}
	}
}

func TestNewTFRegistryProviderRelease(t *testing.T) {
	cases := []struct {
		source       string
		namespace    string
		providerType string
		ok           bool
	}{
		{
			source:       "hoge/piyo",
			namespace:    "hoge",
			providerType: "piyo",
			ok:           true,
		},
		{
			source:       "hoge",
			namespace:    "",
			providerType: "",
			ok:           false,
		},
	}

	for _, tc := range cases {
		config := tfregistry.Config{}
		got, err := NewTFRegistryProviderRelease(tc.source, config)

		if tc.ok && err != nil {
			t.Errorf("NewTFRegistryProviderRelease() with source = %s returns unexpected err: %s", tc.source, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewTFRegistryProviderRelease() with source = %s expects to return an error, but no error", tc.source)
		}

		if tc.ok {
			r := got.(*TFRegistryProviderRelease)

			if !(r.namespace == tc.namespace && r.providerType == tc.providerType) {
				t.Errorf("NewTFRegistryProviderRelease() with source = %s returns (%s, %s), but want (%s, %s)", tc.source, r.namespace, r.providerType, tc.namespace, tc.providerType)
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
				moduleRes: &tfregistry.ListModuleVersionsResponse{
					Modules: []tfregistry.ModuleVersions{
						{
							Versions: []tfregistry.ModuleVersion{
								{Version: "0.3.0"},
								{Version: "0.2.0"},
								{Version: "0.1.0"},
							},
						},
					},
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
		config := tfregistry.Config{}
		r, err := NewTFRegistryModuleRelease(source, config)
		if err != nil {
			t.Fatalf("failed to NewTFRegistryModuleRelease(%s, %#v): %s", source, config, err)
		}
		r.(*TFRegistryModuleRelease).api = tc.client

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
				providerRes: &tfregistry.ListProviderVersionsResponse{
					Versions: []tfregistry.ProviderVersion{
						{Version: "0.3.0"},
						{Version: "0.2.0"},
						{Version: "0.1.0"},
					},
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
		config := tfregistry.Config{}
		r, err := NewTFRegistryProviderRelease(source, config)
		if err != nil {
			t.Fatalf("failed to NewTFRegistryProviderRelease(%s, %#v): %s", source, config, err)
		}
		r.(*TFRegistryProviderRelease).api = tc.client

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
