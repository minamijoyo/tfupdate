package tfupdate

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestNewTerraformUpdater(t *testing.T) {
	cases := []struct {
		version string
		want    Updater
		ok      bool
	}{
		{
			version: "0.12.7",
			want: &TerraformUpdater{
				version: "0.12.7",
			},
			ok: true,
		},
		{
			version: "",
			want:    nil,
			ok:      false,
		},
	}

	for _, tc := range cases {
		got, err := NewTerraformUpdater(tc.version)
		if tc.ok && err != nil {
			t.Errorf("NewTerraformUpdater() with version = %s returns unexpected err: %+v", tc.version, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewTerraformUpdater() with version = %s expects to return an error, but no error", tc.version)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewTerraformUpdater() with version = %s returns %#v, but want = %#v", tc.version, got, tc.want)
		}
	}
}

func TestUpdateTerraform(t *testing.T) {
	cases := []struct {
		filename string
		src      string
		version  string
		want     string
		ok       bool
	}{
		{
			filename: "main.tf",
			src: `
terraform {
  required_version = "0.12.6"
}
`,
			version: "0.12.7",
			want: `
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    null = "2.1.1"
  }
}
`,
			version: "0.12.7",
			want: `
terraform {
  required_providers {
    null = "2.1.1"
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
provider "aws" {
  version = "2.11.0"
  region  = "ap-northeast-1"
}
`,
			version: "0.12.7",
			want: `
provider "aws" {
  version = "2.11.0"
  region  = "ap-northeast-1"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `terraform {
  backend "s3" {
    region = "ap-northeast-1"
    bucket = "hoge"
    key    = "terraform.tfstate"
  }
}
terraform {
  required_version = "0.12.6"
}
`,
			version: "0.12.7",
			want: `terraform {
  backend "s3" {
    region = "ap-northeast-1"
    bucket = "hoge"
    key    = "terraform.tfstate"
  }
}
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
		{
			filename: ".terraform.lock.hcl",
			src: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.1.1"
  constraints = "3.1.1"
  hashes = [
    "h1:YvH6gTaQzGdNv+SKTZujU1O0bO+Pw6vJHOPhqgN8XNs=",
    "zh:063466f41f1d9fd0dd93722840c1314f046d8760b1812fa67c34de0afcba5597",
  ]
}

`,
			version: "0.12.7",
			want: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.1.1"
  constraints = "3.1.1"
  hashes = [
    "h1:YvH6gTaQzGdNv+SKTZujU1O0bO+Pw6vJHOPhqgN8XNs=",
    "zh:063466f41f1d9fd0dd93722840c1314f046d8760b1812fa67c34de0afcba5597",
  ]
}

`,
			ok: true,
		},
	}

	for _, tc := range cases {
		u := &TerraformUpdater{
			version: tc.version,
		}
		f, diags := hclwrite.ParseConfig([]byte(tc.src), tc.filename, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %s", diags)
		}

		err := u.Update(nil, tc.filename, f)
		if tc.ok && err != nil {
			t.Errorf("Update() with src = %s, version = %s returns unexpected err: %+v", tc.src, tc.version, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("Update() with src = %s, version = %s expects to return an error, but no error", tc.src, tc.version)
		}

		got := string(hclwrite.Format(f.BuildTokens(nil).Bytes()))
		if got != tc.want {
			t.Errorf("Update() with src = %s, version = %s returns %s, but want = %s", tc.src, tc.version, got, tc.want)
		}
	}
}
