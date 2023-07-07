package tfupdate

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestNewModuleUpdater(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    Updater
		ok      bool
	}{
		{
			name:    "terraform-aws-modules/vpc/aws",
			version: "2.17.0",
			want: &ModuleUpdater{
				name:    "terraform-aws-modules/vpc/aws",
				version: "2.17.0",
			},
			ok: true,
		},
		{
			name:    "",
			version: "2.17.0",
			want:    nil,
			ok:      false,
		},
		{
			name:    "terraform-aws-modules/vpc/aws",
			version: "",
			want:    nil,
			ok:      false,
		},
	}

	for _, tc := range cases {
		got, err := NewModuleUpdater(tc.name, tc.version)
		if tc.ok && err != nil {
			t.Errorf("NewModuleUpdater() with name = %s, version = %s returns unexpected err: %+v", tc.name, tc.version, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewModuleUpdater() with name = %s, version = %s expects to return an error, but no error", tc.name, tc.version)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewModuleUpdater() with name = %s, version = %s returns %#v, but want = %#v", tc.name, tc.version, got, tc.want)
		}
	}
}

func TestUpdateModule(t *testing.T) {
	cases := []struct {
		filename string
		src      string
		name     string
		version  string
		want     string
		ok       bool
	}{
		{
			filename: "main.tf",
			src: `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.17.0"
}
`,
			name:    "terraform-aws-modules/vpc/aws",
			version: "2.18.0",
			want: `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.18.0"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
module "vpc1" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.17.0"
}
module "vpc2" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.17.0"
}
`,
			name:    "terraform-aws-modules/vpc/aws",
			version: "2.18.0",
			want: `
module "vpc1" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.18.0"
}
module "vpc2" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.18.0"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.17.0"
}
`,
			name:    "terraform-aws-modules/hoge/aws",
			version: "2.18.0",
			want: `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.17.0"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`,
			name:    "terraform-aws-modules/vpc/aws",
			version: "2.18.0",
			want: `
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1.2.0"
}
`,
			name:    "git::https://example.com/vpc.git",
			version: "1.3.0",
			want: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1.3.0"
}
`,
			ok: true,
		},
	}

	for _, tc := range cases {
		u := &ModuleUpdater{
			name:    tc.name,
			version: tc.version,
		}
		f, diags := hclwrite.ParseConfig([]byte(tc.src), tc.filename, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %s", diags)
		}

		err := u.Update(context.Background(), nil, tc.filename, f)
		if tc.ok && err != nil {
			t.Errorf("Update() with src = %s, name = %s, version = %s returns unexpected err: %+v", tc.src, tc.name, tc.version, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("Update() with src = %s, name = %s, version = %s expects to return an error, but no error", tc.src, tc.name, tc.version)
		}

		got := string(hclwrite.Format(f.BuildTokens(nil).Bytes()))
		if got != tc.want {
			t.Errorf("Update() with src = %s, name = %s, version = %s returns %s, but want = %s", tc.src, tc.name, tc.version, got, tc.want)
		}
	}
}

func TestParseModuleSource(t *testing.T) {
	cases := []struct {
		src     string
		name    string
		version string
	}{
		{
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git"
}
`,
			name:    "git::https://example.com/vpc.git",
			version: "",
		},
		{
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1"
}
`,
			name:    "git::https://example.com/vpc.git",
			version: "1",
		},
		{
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1.2"
}
`,
			name:    "git::https://example.com/vpc.git",
			version: "1.2",
		},
		{
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1.2.0"
}
`,
			name:    "git::https://example.com/vpc.git",
			version: "1.2.0",
		},
		{
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v1.2.0-rc1"
}
`,
			name:    "git::https://example.com/vpc.git",
			version: "1.2.0-rc1",
		},
		{
			src: `
module "vpc" {
  source = "git::https://example.com/vpc.git?ref=vhoge"
}
`,
			name:    "git::https://example.com/vpc.git?ref=vhoge",
			version: "",
		},
	}

	for _, tc := range cases {
		f, diags := hclwrite.ParseConfig([]byte(tc.src), "", hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %s", diags)
		}

		m := allMatchingBlocksByType(f.Body(), "module")
		if len(m) != 1 {
			t.Fatalf("failed to get module block: %s", tc.src)
		}
		s := m[0].Body().GetAttribute("source")
		if s == nil {
			t.Fatalf("failed to get module source attribute: %s", tc.src)
		}
		name, version := parseModuleSource(s)

		if !(name == tc.name && version == tc.version) {
			t.Errorf("parseModuleSource() with src = %s returns (%s, %s), but want = (%s, %s)", tc.src, name, version, tc.name, tc.version)
		}
	}
}
