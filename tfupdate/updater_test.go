package tfupdate

import (
	"reflect"
	"testing"
)

func TestNewUpdater(t *testing.T) {
	cases := []struct {
		updaterType string
		name        string
		version     string
		want        Updater
		ok          bool
	}{
		{
			updaterType: "terraform",
			name:        "",
			version:     "0.12.7",
			want: &TerraformUpdater{
				version: "0.12.7",
			},
			ok: true,
		},
		{
			updaterType: "provider",
			name:        "aws",
			version:     "2.23.0",
			want: &ProviderUpdater{
				name:    "aws",
				version: "2.23.0",
			},
			ok: true,
		},
		{
			updaterType: "module",
			name:        "terraform-aws-modules/vpc/aws",
			version:     "2.14.0",
			want:        nil,
			ok:          false,
		},
		{
			updaterType: "hoge",
			name:        "",
			version:     "0.0.1",
			want:        nil,
			ok:          false,
		},
	}

	for _, tc := range cases {
		got, err := NewUpdater(tc.updaterType, tc.name, tc.version)
		if tc.ok && err != nil {
			t.Errorf("NewUpdater() with updateType = %s, name = %s, version = %s returns unexpected err: %+v", tc.updaterType, tc.name, tc.version, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewUpdater() with updateType = %s, name = %s, version = %s expects to return an error, but no error: %+v", tc.updaterType, tc.name, tc.version, err)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewUpdater() with updateType = %s, name = %s, version = %s returns %#v, but want = %#v", tc.updaterType, tc.name, tc.version, got, tc.want)
		}
	}
}
