package tfupdate

import (
	"bytes"
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/minamijoyo/tfupdate/lock"
	"github.com/spf13/afero"
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
				version:    "0.12.7",
			},
			want: &TerraformUpdater{
				version: "0.12.7",
			},
			ok: true,
		},
		{
			o: Option{
				updateType: "provider",
				name:       "aws",
				version:    "2.23.0",
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
				name:       "terraform-aws-modules/vpc/aws",
				version:    "2.14.0",
			},
			want: &ModuleUpdater{
				name:    "terraform-aws-modules/vpc/aws",
				version: "2.14.0",
			},
			ok: true,
		},
		{
			o: Option{
				updateType: "lock",
				platforms:  []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			},
			want: &LockUpdater{
				platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			},
			ok: true,
		},
		{
			o: Option{
				updateType: "hoge",
				version:    "0.0.1",
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
			t.Errorf("NewUpdater() with o = %#v expects to return an error, but no error", tc.o)
		}

		opts := []cmp.Option{
			cmp.AllowUnexported(TerraformUpdater{}),
			cmp.AllowUnexported(ProviderUpdater{}),
			cmp.AllowUnexported(ModuleUpdater{}),
			cmp.AllowUnexported(LockUpdater{}),
			cmpopts.IgnoreInterfaces(struct{ lock.Index }{}),
		}
		if diff := cmp.Diff(got, tc.want, opts...); diff != "" {
			t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
		}
	}
}

func TestUpdateHCL(t *testing.T) {
	cases := []struct {
		src       string
		o         Option
		want      string
		isUpdated bool
		ok        bool
	}{
		{
			src: `
terraform {
  required_version = "0.12.4"
}
`,
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
			},
			// Note the lack of space here.
			// the current implementation of (*hclwrite.Body).SetAttributeValue()
			// does not seem to preserve an original SpaceBefore value of attribute.
			// This is a bug of upstream.
			// We avoid this by formating the output of this function.
			want: `
terraform {
  required_version ="0.12.7"
}
`,
			isUpdated: true,
			ok:        true,
		},
		{
			src: `
provider "aws" {
  version = "2.11.0"
}
`,
			o: Option{
				updateType: "provider",
				name:       "aws",
				version:    "2.23.0",
			},
			want: `
provider "aws" {
  version ="2.23.0"
}
`,
			isUpdated: true,
			ok:        true,
		},
		{
			src: `
provider "aws" {
  version = "2.11.0"
}
`,
			o: Option{
				updateType: "provider",
				name:       "hoge",
				version:    "2.23.0",
			},
			want: `
provider "aws" {
  version = "2.11.0"
}
`,
			isUpdated: false,
			ok:        true,
		},
		{
			src: `
provider "invalid" {
`,
			o: Option{
				updateType: "provider",
				name:       "hoge",
				version:    "2.23.0",
			},
			want:      "",
			isUpdated: false,
			ok:        false,
		},
		{
			// not panic even if a map index is a variable reference
			src: `resource "not_panic" "hoge" {
  b = a[var.env]
}
`,
			o: Option{
				updateType: "provider",
				name:       "hoge",
				version:    "2.23.0",
			},
			want: `resource "not_panic" "hoge" {
  b = a[var.env]
}
`,
			isUpdated: false,
			ok:        true,
		},
	}

	for _, tc := range cases {
		r := bytes.NewBufferString(tc.src)
		w := &bytes.Buffer{}

		fs := afero.NewMemMapFs()
		gc, err := NewGlobalContext(fs, tc.o)
		if err != nil {
			t.Fatalf("failed to new global context: %s", err)
		}

		mc, err := NewModuleContext(".", gc)
		if err != nil {
			t.Fatalf("failed to new module context: %s", err)
		}

		isUpdated, err := UpdateHCL(context.Background(), mc, r, w, "main.tf")
		if tc.ok && err != nil {
			t.Errorf("UpdateHCL() with src = %s, o = %#v returns unexpected err: %+v", tc.src, tc.o, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("UpdateHCL() with src = %s, o = %#v expects to return an error, but no error", tc.src, tc.o)
		}

		if isUpdated != tc.isUpdated {
			t.Errorf("UpdateHCL() with src = %s, o = %#v expects to return isUpdated = %t, but want = %t", tc.src, tc.o, isUpdated, tc.isUpdated)
		}

		got := w.String()
		if got != tc.want {
			t.Errorf("UpdateHCL() with src = %s, o = %#v returns %s, but want = %s", tc.src, tc.o, got, tc.want)
		}
	}
}
