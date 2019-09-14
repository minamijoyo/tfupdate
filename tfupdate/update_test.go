package tfupdate

import (
	"bytes"
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
				updateType: "terraform",
				target:     "0.12.7",
			},
			want: &TerraformUpdater{
				version: "0.12.7",
			},
			ok: true,
		},
		{
			o: Option{
				updateType: "provider",
				target:     "aws@2.23.0",
			},
			want: &ProviderUpdater{
				name:    "aws",
				version: "2.23.0",
			},
			ok: true,
		},
		{
			o: Option{
				updateType: "module",
				target:     "terraform-aws-modules/vpc/aws@2.14.0",
			},
			want: nil,
			ok:   false,
		},
		{
			o: Option{
				updateType: "hoge",
				target:     "0.0.1",
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

func TestUpdateHCL(t *testing.T) {
	cases := []struct {
		src  string
		o    Option
		want string
		ok   bool
	}{
		{
			src: `
terraform {
  required_version = "0.12.4"
}
`,
			o: Option{
				updateType: "terraform",
				target:     "0.12.7",
			},
			want: `
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
		{
			src: `
provider "aws" {
  version = "2.11.0"
}
`,
			o: Option{
				updateType: "provider",
				target:     "aws@2.23.0",
			},
			want: `
provider "aws" {
  version = "2.23.0"
}
`,
			ok: true,
		},
		{
			src: `
provider "aws" {
  version = "2.11.0"
}
`,
			o: Option{
				updateType: "provider",
				target:     "hoge@2.23.0",
			},
			want: "",
			ok:   true,
		},
		{
			src: `
provider "invalid" {
`,
			o: Option{
				updateType: "provider",
				target:     "hoge@2.23.0",
			},
			want: "",
			ok:   false,
		},
		{
			src: `
provider "aws" {
  version = "2.11.0"
}
`,
			o: Option{
				updateType: "hoge",
				target:     "0.0.1",
			},
			want: "",
			ok:   false,
		},
	}

	for _, tc := range cases {
		r := bytes.NewBufferString(tc.src)
		w := &bytes.Buffer{}
		err := UpdateHCL(r, w, "test", tc.o)
		if tc.ok && err != nil {
			t.Errorf("UpdateHCL() with src = %s, o = %#v returns unexpected err: %+v", tc.src, tc.o, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("UpdateHCL() with src = %s, o = %#v expects to return an error, but no error: %+v", tc.src, tc.o, err)
		}

		got := string(w.Bytes())
		if got != tc.want {
			t.Errorf("UpdateHCL() with src = %s, o = %#v returns %s, but want = %s", tc.src, tc.o, got, tc.want)
		}
	}
}
