package tfupdate

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/spf13/afero"
)

func TestUpdateFileExist(t *testing.T) {
	cases := []struct {
		filename string
		src      string
		o        Option
		want     string
		ok       bool
	}{
		{
			filename: "valid.tf",
			src: `
terraform {
  required_version = "0.12.6"
}
`,
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
			},
			want: `
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
		{
			filename: "unformatted_match.tf",
			src: `
terraform {
required_version = "0.12.6"
}
`,
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
			},
			want: `
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
		{
			filename: "unformatted_mo_match.tf",
			src: `
terraform {
required_version = "0.12.6"
}
`,
			o: Option{
				updateType: "provider",
				name:       "aws",
				version:    "2.23.0",
			},
			want: `
terraform {
required_version = "0.12.6"
}
`,
			ok: true,
		},
		{
			filename: "valid.hcl",
			src: `
terraform {
  required_version = "0.12.6"
}
`,
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
			},
			want: `
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
	}
	for _, tc := range cases {
		fs := afero.NewMemMapFs()
		err := afero.WriteFile(fs, tc.filename, []byte(tc.src), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %s", err)
		}

		gc, err := NewGlobalContext(fs, tc.o)
		if err != nil {
			t.Fatalf("failed to new global context: %s", err)
		}

		mc, err := NewModuleContext(".", gc)
		if err != nil {
			t.Fatalf("failed to new module context: %s", err)
		}

		err = UpdateFile(context.Background(), mc, tc.filename)
		if tc.ok && err != nil {
			t.Errorf("UpdateFile() with filename = %s, o = %#v returns unexpected err: %+v", tc.filename, tc.o, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("UpdateFile() with filename = %s, o = %#v expects to return an error, but no error", tc.filename, tc.o)
		}

		got, err := afero.ReadFile(fs, tc.filename)
		if err != nil {
			t.Fatalf("failed to read updated file: %s", err)
		}

		if string(got) != tc.want {
			t.Errorf("UpdateFile() with filename = %s, o = %#v returns %s, but want = %s", tc.filename, tc.o, string(got), tc.want)
		}
	}
}

func TestUpdateFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	filename := "not_found.tf"
	o := Option{
		updateType: "terraform",
		version:    "0.12.7",
	}

	gc, err := NewGlobalContext(fs, o)
	if err != nil {
		t.Fatalf("failed to new global context: %s", err)
	}

	mc, err := NewModuleContext(".", gc)
	if err != nil {
		t.Fatalf("failed to new module context: %s", err)
	}

	err = UpdateFile(context.Background(), mc, filename)

	if err == nil {
		t.Errorf("UpdateFile() with filename = %s, o = %#v expects to return an error, but no error", filename, o)
	}
}

func TestUpdateDirExist(t *testing.T) {
	cases := []struct {
		rootdir   string
		subdir    string
		filename1 string
		src1      string
		filename2 string
		src2      string
		o         Option
		checkdir  string
		want1     string
		want2     string
	}{
		{
			rootdir:   "a",
			subdir:    "b",
			filename1: "terraform.tf",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "provider.tf",
			src2: `
provider "aws" {
  version = "2.11.0"
}
`,
			checkdir: "a/b",
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
				recursive:  false,
			},
			want1: `
terraform {
  required_version = "0.12.7"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
}
`,
		},
		{
			rootdir:   "a",
			subdir:    "b",
			filename1: "terraform.tf",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "provider.tf",
			src2: `
provider "aws" {
  version = "2.11.0"
}
`,
			checkdir: "a",
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
				recursive:  true,
			},
			want1: `
terraform {
  required_version = "0.12.7"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
}
`,
		},
		{
			rootdir:   "a",
			subdir:    "b",
			filename1: "terraform.tf",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "provider.tf",
			src2: `
provider "aws" {
  version = "2.11.0"
}
`,
			checkdir: "a",
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
				recursive:  false,
			},
			want1: `
terraform {
  required_version = "0.12.6"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
}
`,
		},
		{
			rootdir:   "a",
			subdir:    ".terraform",
			filename1: "terraform.tf",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "provider.tf",
			src2: `
provider "aws" {
  version = "2.11.0"
}
`,
			checkdir: "a",
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
				recursive:  true,
			},
			want1: `
terraform {
  required_version = "0.12.6"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
}
`,
		},
		{
			rootdir:   "a",
			subdir:    ".git",
			filename1: "terraform.tf",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "provider.tf",
			src2: `
provider "aws" {
  version = "2.11.0"
}
`,
			checkdir: "a",
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
				recursive:  true,
			},
			want1: `
terraform {
  required_version = "0.12.6"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
}
`,
		},
		{
			rootdir:   "a",
			subdir:    "b",
			filename1: "terraform.hcl",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "provider.tf",
			src2: `
provider "aws" {
  version = "2.11.0"
}
`,
			checkdir: "a/b",
			o: Option{
				updateType: "terraform",
				version:    "0.12.7",
				recursive:  false,
			},
			want1: `
terraform {
  required_version = "0.12.6"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
}
`,
		},
		{
			rootdir:   "a",
			subdir:    "b",
			filename1: "terraform.tf",
			src1: `
terraform {
  required_version = "0.12.6"
}
`,
			filename2: "ignore.tf",
			src2: `
terraform {
  required_version = "0.12.6"
}
`,
			checkdir: "a/b",
			o: Option{
				updateType:  "terraform",
				version:     "0.12.7",
				recursive:   false,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile(`a/b/ignore.tf`)},
			},
			want1: `
terraform {
  required_version = "0.12.7"
}
`,
			want2: `
terraform {
  required_version = "0.12.6"
}
`,
		},
	}

	for _, tc := range cases {
		fs := afero.NewMemMapFs()
		dirname := filepath.Join(tc.rootdir, tc.subdir)
		err := fs.MkdirAll(dirname, os.ModePerm)
		if err != nil {
			t.Fatalf("failed to create dir: %s", err)
		}

		err = afero.WriteFile(fs, filepath.Join(dirname, tc.filename1), []byte(tc.src1), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %s", err)
		}

		err = afero.WriteFile(fs, filepath.Join(dirname, tc.filename2), []byte(tc.src2), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %s", err)
		}

		gc, err := NewGlobalContext(fs, tc.o)
		if err != nil {
			t.Fatalf("failed to new global context: %s", err)
		}

		mc, err := NewModuleContext(dirname, gc)
		if err != nil {
			t.Fatalf("failed to new module context: %s", err)
		}

		err = UpdateDir(context.Background(), mc, tc.checkdir)

		if err != nil {
			t.Errorf("UpdateDir() with dirname = %s, o = %#v returns an unexpected error: %+v", tc.checkdir, tc.o, err)
		}

		got1, err := afero.ReadFile(fs, filepath.Join(dirname, tc.filename1))
		if err != nil {
			t.Fatalf("failed to read file: %s", err)
		}

		if string(got1) != tc.want1 {
			t.Errorf("UpdateDir() with dirname = %s, o = %#v returns %s, but want = %s", dirname, tc.o, string(got1), tc.want1)
		}

		got2, err := afero.ReadFile(fs, filepath.Join(dirname, tc.filename2))
		if err != nil {
			t.Fatalf("failed to read file: %s", err)
		}

		if string(got2) != tc.want2 {
			t.Errorf("UpdateDir() with dirname = %s, o = %#v returns %s, but want = %s", dirname, tc.o, string(got2), tc.want2)
		}
	}
}

func TestUpdateFileOrDirFile(t *testing.T) {
	src := `
terraform {
  required_version = "0.12.6"
}
`
	o := Option{
		updateType: "terraform",
		version:    "0.12.7",
	}

	cases := []struct {
		desc     string
		filename string
		path     string
		want     string
	}{
		{
			desc:     "simple dir with tf file",
			filename: "terraform.tf",
			path:     "a",
			want: `
terraform {
  required_version = "0.12.7"
}
`,
		},
		{
			desc:     "simple tf file",
			filename: "terraform.tf",
			path:     "a/terraform.tf",
			want: `
terraform {
  required_version = "0.12.7"
}
`,
		},
		{
			desc:     "should not update .hcl file if the target path is dir",
			filename: "terraform.hcl",
			path:     "a",
			want: `
terraform {
  required_version = "0.12.6"
}
`,
		},
		{
			desc:     "should update .hcl file if the target path is file",
			filename: "terraform.hcl",
			path:     "a/terraform.hcl",
			want: `
terraform {
  required_version = "0.12.7"
}
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			dirname := "a"
			err := fs.MkdirAll(dirname, os.ModePerm)
			if err != nil {
				t.Fatalf("failed to create dir: %s", err)
			}

			err = afero.WriteFile(fs, filepath.Join(dirname, tc.filename), []byte(src), 0644)
			if err != nil {
				t.Fatalf("failed to write file: %s", err)
			}

			gc, err := NewGlobalContext(fs, o)
			if err != nil {
				t.Fatalf("failed to new global context: %s", err)
			}

			err = UpdateFileOrDir(context.Background(), gc, tc.path)

			if err != nil {
				t.Errorf("UpdateFileOrDir() with path = %s, o = %#v returns an unexpected error: %+v", tc.path, o, err)
			}

			got, err := afero.ReadFile(fs, filepath.Join(dirname, tc.filename))
			if err != nil {
				t.Fatalf("failed to read file: %s", err)
			}

			if string(got) != tc.want {
				t.Errorf("UpdateFileOrDir() with path = %s, o = %#v returns %s, but want = %s", tc.path, o, string(got), tc.want)
			}
		})
	}
}
