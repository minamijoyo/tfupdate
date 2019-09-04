package tfupdate

import (
	"reflect"
	"testing"
)

func TestNewUpdater(t *testing.T) {
	cases := []struct {
		o    Option
		want Updater
		ok   bool
	}{
		{
			o: Option{
				updaterType: "terraform",
				name:        "",
				version:     "0.12.7",
			},
			want: &TerraformUpdater{
				version: "0.12.7",
			},
			ok: true,
		},
		{
			o: Option{
				updaterType: "provider",
				name:        "aws",
				version:     "2.23.0",
			},
			want: &ProviderUpdater{
				name:    "aws",
				version: "2.23.0",
			},
			ok: true,
		},
		{
			o: Option{
				updaterType: "module",
				name:        "terraform-aws-modules/vpc/aws",
				version:     "2.14.0",
			},
			want: nil,
			ok:   false,
		},
		{
			o: Option{
				updaterType: "hoge",
				name:        "",
				version:     "0.0.1",
			},
			want: nil,
			ok:   false,
		},
	}

	for _, tc := range cases {
		got, err := NewUpdater(tc.o)
		if tc.ok && err != nil {
			t.Errorf("NewUpdater() with o = %#v returns unexpected err: %+v", tc.o, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewUpdater() with o = %#v expects to return an error, but no error: %+v", tc.o, err)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewUpdater() with o = %#v returns %#v, but want = %#v", tc.o, got, tc.want)
		}
	}
}
