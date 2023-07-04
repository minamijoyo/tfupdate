package lock

import (
	"context"
	"testing"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

// mockTFRegistryClient is a mock TFRegistryAPI implementation.
type mockTFRegistryClient struct {
	metadataRes *tfregistry.ProviderPackageMetadataResponse
	err         error
}

var _ TFRegistryAPI = (*mockTFRegistryClient)(nil)

func (c *mockTFRegistryClient) ProviderPackageMetadata(ctx context.Context, req *tfregistry.ProviderPackageMetadataRequest) (*tfregistry.ProviderPackageMetadataResponse, error) { // nolint revive unused-parameter
	return c.metadataRes, c.err
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
