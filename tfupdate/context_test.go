package tfupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

func TestModuleContextSelecetedProviders(t *testing.T) {
	cases := []struct {
		desc string
		src  string
		want []SelectedProvider
	}{
		{
			desc: "simple",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "3.1.1"
    }

    github = {
      source  = "integrations/github"
      version = "4.28.0"
    }
  }
}
`,
			want: []SelectedProvider{
				SelectedProvider{Source: "hashicorp/aws", Version: "5.4.0"},
				SelectedProvider{Source: "hashicorp/null", Version: "3.1.1"},
				SelectedProvider{Source: "integrations/github", Version: "4.28.0"},
			},
		},
		{
			desc: "empty",
			src: `
terraform {
  required_version = "1.5.0"
}
`,
			want: []SelectedProvider{},
		},
		{
			desc: "unknown source",
			src: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      version = "3.1.1"
    }
  }
}
`,
			want: []SelectedProvider{
				SelectedProvider{Source: "hashicorp/aws", Version: "5.4.0"},
			},
		},
		{
			desc: "unknown version",
			src: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
    }
  }
}
`,
			want: []SelectedProvider{
				SelectedProvider{Source: "hashicorp/aws", Version: "5.4.0"},
			},
		},
		{
			desc: "provider block (unknown source)",
			src: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }
  }
}

provider "null" {
  version = "3.2.1"
}
`,
			want: []SelectedProvider{
				SelectedProvider{Source: "hashicorp/aws", Version: "5.4.0"},
			},
		},
		{
			desc: "version constraint",
			src: `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "> 3.0.0"
    }
  }
}
`,
			want: []SelectedProvider{
				SelectedProvider{Source: "hashicorp/aws", Version: "5.4.0"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
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

			gc := &GlobalContext{
				fs: fs,
			}
			mc, err := NewModuleContext(dirname, gc)
			if err != nil {
				t.Fatalf("failed to new ModuleContext: %s", err)
			}

			got := mc.SelecetedProviders()

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
			}
		})
	}
}

func TestSelectVersion(t *testing.T) {
	cases := []struct {
		desc        string
		constraints []string
		want        string
	}{
		{
			desc:        "simple",
			constraints: []string{"3.2.1"},
			want:        "3.2.1",
		},
		{
			desc:        "empty list",
			constraints: []string{},
			want:        "",
		},
		{
			desc:        "empty string",
			constraints: []string{""},
			want:        "",
		},
		{
			desc:        "return first one found",
			constraints: []string{"1.2.3", "3.2.1"},
			want:        "1.2.3",
		},
		{
			desc:        "ignore parse error",
			constraints: []string{"> 1.2.3"},
			want:        "",
		},
		{
			desc:        "ignore parse error and return first one found",
			constraints: []string{"= 1.2.3", "3.2.1"},
			want:        "3.2.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := selectVersion(tc.constraints)

			if got != tc.want {
				t.Errorf("got=%s, but want=%s", got, tc.want)
			}
		})
	}
}

func TestModuleContextResolveProviderShortNameFromSource(t *testing.T) {
	cases := []struct {
		desc   string
		src    string
		source string
		want   string
	}{
		{
			desc: "simple",
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
			source: "integrations/github",
			want:   "github",
		},
		{
			desc: "multiple forks",
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
			source: "winebarrel/mysql",
			want:   "winebarrel",
		},
		{
			desc: "not found",
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
			source: "foo/mysql",
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
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

			gc := &GlobalContext{
				fs: fs,
			}
			mc, err := NewModuleContext(dirname, gc)
			if err != nil {
				t.Fatalf("failed to new ModuleContext: %s", err)
			}

			got := mc.ResolveProviderShortNameFromSource(tc.source)

			if got != tc.want {
				t.Errorf("got: %s, want = %s", got, tc.want)
			}
		})
	}
}
