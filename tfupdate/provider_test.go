package tfupdate

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/afero"
)

func TestNewProviderUpdater(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    Updater
		ok      bool
	}{
		{
			name:    "aws",
			version: "2.23.0",
			want: &ProviderUpdater{
				name:    "aws",
				version: "2.23.0",
			},
			ok: true,
		},
		{
			name:    "",
			version: "2.23.0",
			want:    nil,
			ok:      false,
		},
		{
			name:    "aws",
			version: "",
			want:    nil,
			ok:      false,
		},
	}

	for _, tc := range cases {
		got, err := NewProviderUpdater(tc.name, tc.version)
		if tc.ok && err != nil {
			t.Errorf("NewProviderUpdater() with name = %s, version = %s returns unexpected err: %+v", tc.name, tc.version, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewProviderUpdater() with name = %s, version = %s expects to return an error, but no error", tc.name, tc.version)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewProviderUpdater() with name = %s, version = %s returns %#v, but want = %#v", tc.name, tc.version, got, tc.want)
		}
	}
}

func TestUpdateProvider(t *testing.T) {
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
terraform {
  required_version = "0.12.4"
  required_providers {
    null = "2.1.1"
  }
}
`,
			name:    "null",
			version: "2.1.2",
			want: `
terraform {
  required_version = "0.12.4"
  required_providers {
    null = "2.1.2"
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
			name:    "aws",
			version: "2.23.0",
			want: `
provider "aws" {
  version = "2.23.0"
  region  = "ap-northeast-1"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_version = "0.12.4"
  required_providers {
    null = "2.1.1"
  }
}
`,
			name:    "aws",
			version: "2.23.0",
			want: `
terraform {
  required_version = "0.12.4"
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
  region = "ap-northeast-1"
}
`,
			name:    "aws",
			version: "2.23.0",
			want: `
provider "aws" {
  region = "ap-northeast-1"
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
terraform {
  required_providers {
    aws = "2.11.0"
  }
}
provider "aws" {
  alias   = "one"
  version = "2.11.0"
  region  = "ap-northeast-1"
}
provider "aws" {
  alias   = "two"
  version = "2.11.0"
  region  = "us-east-1"
}
`,
			name:    "aws",
			version: "2.23.0",
			want: `
terraform {
  required_providers {
    null = "2.1.1"
  }
}
terraform {
  required_providers {
    aws = "2.23.0"
  }
}
provider "aws" {
  alias   = "one"
  version = "2.23.0"
  region  = "ap-northeast-1"
}
provider "aws" {
  alias   = "two"
  version = "2.23.0"
  region  = "us-east-1"
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "2.65.0"
    }
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "2.66.0"
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "2.1.2"
    }

    aws = {
      source  = "hashicorp/aws"
      version = "2.65.0"
    }
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "2.1.2"
    }

    aws = {
      source  = "hashicorp/aws"
      version = "2.66.0"
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    # foo
    aws = "2.65.0" # bar
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    # foo
    aws = "2.66.0" # bar
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    # foo
    aws = {
      # version = "2.65.0" # bar
      version = "2.65.0" # baz
      source  = "hashicorp/aws"
    }
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    # foo
    aws = {
      # version = "2.65.0" # bar
      version = "2.66.0" # baz
      source  = "hashicorp/aws"
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			// a TokenQuotedLit is also valid token for version
			src: `
terraform {
  required_providers {
    aws = {
      "version" = "2.65.0"
      "source"  = "hashicorp/aws"
    }
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    aws = {
      "version" = "2.66.0"
      "source"  = "hashicorp/aws"
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    aws = {
      version = "2.65.0"
      source  = "hashicorp/aws"

      configuration_aliases = [
        aws.primary,
        aws.secondary,
      ]
    }
  }
}
`,
			name:    "aws",
			version: "2.66.0",
			want: `
terraform {
  required_providers {
    aws = {
      version = "2.66.0"
      source  = "hashicorp/aws"

      configuration_aliases = [
        aws.primary,
        aws.secondary,
      ]
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.38.0"
    }
  }
}
`,
			name:    "integrations/github",
			version: "5.39.0",
			want: `
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.39.0"
    }
  }
}
`,
			ok: true,
		},
		{
			filename: "main.tf",
			src: `
terraform {
  required_providers {
    petoju = {
      source  = "petoju/mysql"
      version = "3.0.41"
    }

    winebarrel = {
      source  = "winebarrel/mysql"
      version = "1.10.5"
    }
  }
}
`,
			name:    "winebarrel/mysql",
			version: "1.10.6",
			want: `
terraform {
  required_providers {
    petoju = {
      source  = "petoju/mysql"
      version = "3.0.41"
    }

    winebarrel = {
      source  = "winebarrel/mysql"
      version = "1.10.6"
    }
  }
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
			name:    "null",
			version: "3.2.1",
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
		fs := afero.NewMemMapFs()
		dirname := "test"
		err := fs.MkdirAll(dirname, os.ModePerm)
		if err != nil {
			t.Fatalf("failed to create dir: %s", err)
		}

		err = afero.WriteFile(fs, filepath.Join(dirname, "main.tf"), []byte(tc.src), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %s", err)
		}

		o := Option{
			updateType: "provider",
			name:       tc.name,
			version:    tc.version,
		}
		gc, err := NewGlobalContext(fs, o)
		if err != nil {
			t.Fatalf("failed to new global context: %s", err)
		}

		mc, err := NewModuleContext(dirname, gc)
		if err != nil {
			t.Fatalf("failed to new module context: %s", err)
		}

		u := gc.updater
		f, diags := hclwrite.ParseConfig([]byte(tc.src), tc.filename, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			t.Fatalf("unexpected diagnostics: %s", diags)
		}

		err = u.Update(context.Background(), mc, tc.filename, f)
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
