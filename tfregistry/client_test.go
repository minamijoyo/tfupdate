package tfregistry

import "testing"

func TestNewClient(t *testing.T) {
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
		config := Config{
			BaseURL: tc.baseURL,
		}
		got, err := NewClient(config)

		if tc.ok && err != nil {
			t.Errorf("NewClient() with baseURL = %s returns unexpected err: %s", tc.baseURL, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewClient() with baseURL = %s expects to return an error, but no error", tc.baseURL)
		}

		if tc.ok {
			if got.BaseURL.String() != tc.want {
				t.Errorf("NewClient() with baseURL = %s returns %s, but want %s", tc.baseURL, got.BaseURL.String(), tc.want)
			}
		}
	}
}
