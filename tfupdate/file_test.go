package tfupdate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"
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
		{
			rootdir:   "a",
			subdir:    "b",
			filename1: "terraform.tofu",
			src1: `
terraform {
  required_version = "1.8.0"
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
				updateType: "opentofu",
				version:    "1.9.0",
				recursive:  false,
			},
			want1: `
terraform {
  required_version = "1.9.0"
}
`,
			want2: `
provider "aws" {
  version = "2.11.0"
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

// These CLI tests deliberately make no assumptions about a new Go API.
func buildVersionCLI(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("TFUPDATE_TEST_BINARY"); bin != "" {
		return bin
	}
	bin := filepath.Join(t.TempDir(), "tfupdate")
	args := []string{"build", "-buildvcs=false", "-o", bin}
	if os.Getenv("TFUPDATE_COVER_DIR") != "" {
		args = append(args, "-cover", "-coverpkg=github.com/minamijoyo/tfupdate/...")
	}
	args = append(args, "..")
	cmd := exec.Command("go", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return bin
}

func runVersionCLI(bin, dir string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	if cover := os.Getenv("TFUPDATE_COVER_DIR"); cover != "" {
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+cover)
	}
	return cmd.CombinedOutput()
}

func writeVersionFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func readVersionFixture(t *testing.T, dir, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

func versionValues(t *testing.T, src, name string) ([]string, int) {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(src), name, hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("invalid output %s: %s", name, diags)
	}
	var values []string
	blocks := 0
	for _, block := range f.Body.(*hclsyntax.Body).Blocks {
		if block.Type != "terraform" || len(block.Labels) != 0 {
			continue
		}
		blocks++
		if attr := block.Body.Attributes["required_version"]; attr != nil {
			v, diags := attr.Expr.Value(nil)
			if diags.HasErrors() || v.IsNull() || !v.IsKnown() || v.Type() != cty.String {
				t.Fatalf("%s: expected literal string: %s", name, diags)
			}
			values = append(values, v.AsString())
		}
	}
	return values, blocks
}

func requireAddMissing(t *testing.T, bin, command string) {
	t.Helper()
	dir := t.TempDir()
	out, err := runVersionCLI(bin, dir, command, "--add-missing", "-v", "probe-value", ".")
	if err != nil {
		t.Fatalf("add-missing support prerequisite: %v\n%s", err, out)
	}
	values, _ := versionValues(t, readVersionFixture(t, dir, "versions.tf"), "versions.tf")
	if fmt.Sprint(values) != "[probe-value]" {
		t.Fatalf("feature prerequisite got %q", values)
	}
}

func TestAddMissing(t *testing.T) {
	bin := buildVersionCLI(t)
	const old = "terraform { required_version = \"old\" }\n"
	const keep = "# preserve this comment\nlocals { untouched=\"value\" }\n"
	type scenario struct {
		name, path, version            string
		files                          map[string]string
		dirs, flags, unchanged, absent []string
		want                           map[string]int
		blocks                         map[string]int
		snippets                       map[string][]string
	}
	cases := []scenario{
		{name: "ignore_direct_directory_empty", path: "module", dirs: []string{"module"}, flags: []string{"-i", "module$"}, absent: []string{"module/versions.tf"}},
		{name: "ignore_direct_directory_recursive", path: "module", flags: []string{"-r", "-i", "module$"}, files: map[string]string{"module/main.tf": old, "module/child/main.tofu": keep}, unchanged: []string{"module/main.tf", "module/child/main.tofu"}, absent: []string{"module/versions.tf", "module/child/versions.tf"}},
		{name: "explicit_hcl_native_sibling_suppresses_tf", path: "custom.hcl", files: map[string]string{"custom.hcl": keep, "other.tf": old}, unchanged: []string{"custom.hcl", "other.tf"}, absent: []string{"versions.tf"}},
		{name: "explicit_hcl_native_sibling_suppresses_tofu", path: "custom.hcl", files: map[string]string{"custom.hcl": keep, "other.tofu": old}, unchanged: []string{"custom.hcl", "other.tofu"}, absent: []string{"versions.tf"}},
		{name: "explicit_nonnative_sibling_existing_tf", path: "target.tf", files: map[string]string{"custom.hcl": old, "target.tf": keep}, want: map[string]int{"target.tf": 1}, unchanged: []string{"custom.hcl"}, absent: []string{"versions.tf"}},
		{name: "explicit_nonnative_sibling_missing_tf", path: "target.tf", files: map[string]string{"custom.hcl": old}, want: map[string]int{"target.tf": 1}, unchanged: []string{"custom.hcl"}, absent: []string{"versions.tf"}},
		{name: "hidden_native_tf", files: map[string]string{".constraints.tf": old}, want: map[string]int{".constraints.tf": 1}, absent: []string{"versions.tf"}},
		{name: "explicit_nonnative_sibling_existing_tofu", path: "target.tofu", files: map[string]string{"custom.hcl": old, "target.tofu": keep}, want: map[string]int{"target.tofu": 1}, unchanged: []string{"custom.hcl"}, absent: []string{"versions.tf"}},
		{name: "explicit_nonnative_sibling_missing_tofu", path: "target.tofu", files: map[string]string{"custom.hcl": old}, want: map[string]int{"target.tofu": 1}, unchanged: []string{"custom.hcl"}, absent: []string{"versions.tf"}},
		{name: "hidden_native_tofu", files: map[string]string{".constraints.tofu": old}, want: map[string]int{".constraints.tofu": 1}, absent: []string{"versions.tf"}},
		{name: "ordinary_hcl_not_scanned", files: map[string]string{"ordinary.hcl": old}, want: map[string]int{"versions.tf": 1}, unchanged: []string{"ordinary.hcl"}},
		{name: "comments_on_update", files: map[string]string{"main.tf": "# leading update comment\nterraform {\n # attribute comment\n required_version = \"old\" # inline update comment\n # trailing update comment\n}\n# final update comment\nlocals { value = \"preserved-value\" }\n"}, want: map[string]int{"main.tf": 1}, absent: []string{"versions.tf"}, snippets: map[string][]string{"main.tf": {"# leading update comment", "# attribute comment", "# inline update comment", "# trailing update comment", "# final update comment", "\"preserved-value\""}}},

		{name: "mixed_block_update", files: map[string]string{"main.tf": old + "terraform {}\nterraform \"label\" { required_version = \"label-value\" }\nlocals {\n terraform { required_version = \"nested-value\" }\n}\nrequired_version = \"root-value\"\n" + old}, want: map[string]int{"main.tf": 2}, blocks: map[string]int{"main.tf": 3}, absent: []string{"versions.tf"}, snippets: map[string][]string{"main.tf": {"\"label-value\"", "\"nested-value\"", "\"root-value\""}}},

		{name: "unformatted_same_bytes", version: "old", files: map[string]string{"main.tf": "terraform {\nrequired_version =\"old\"\n}\n"}, want: map[string]int{"main.tf": 1}, absent: []string{"versions.tf"}},

		{name: "already_canonical", version: "old", files: map[string]string{"main.tf": "terraform {\n  required_version = \"old\"\n}\n"}, want: map[string]int{"main.tf": 1}, unchanged: []string{"main.tf"}, absent: []string{"versions.tf"}},
		{name: "single_line_comment", files: map[string]string{"versions.tf": "# before\nterraform { experiments = [] } # after\n"}, want: map[string]int{"versions.tf": 1}, blocks: map[string]int{"versions.tf": 1}, snippets: map[string][]string{"versions.tf": {"# before", "# after", "experiments"}}},

		{name: "empty_directory", want: map[string]int{"versions.tf": 1}},
		{name: "unrelated_file", files: map[string]string{"main.tf": keep}, unchanged: []string{"main.tf"}, want: map[string]int{"versions.tf": 1}},
		{name: "empty_destination", files: map[string]string{"versions.tf": ""}, want: map[string]int{"versions.tf": 1}},
		{name: "reuse_destination_block", files: map[string]string{"versions.tf": "# head\nterraform {\n # inside\n backend \"local\" {}\n}\n"}, want: map[string]int{"versions.tf": 1}, blocks: map[string]int{"versions.tf": 1}, snippets: map[string][]string{"versions.tf": {"# head", "# inside", "backend \"local\""}}},
		{name: "reuse_first_block", files: map[string]string{"versions.tf": "terraform { experiments = [] }\nterraform {}\n"}, want: map[string]int{"versions.tf": 1}, blocks: map[string]int{"versions.tf": 2}},
		{name: "append_after_unrelated", files: map[string]string{"versions.tf": keep}, want: map[string]int{"versions.tf": 1}, snippets: map[string][]string{"versions.tf": {"# preserve this comment", "untouched", "value"}}},
		{name: "declaration_later_file", files: map[string]string{"versions.tf": keep, "z.tofu": old}, want: map[string]int{"z.tofu": 1}, unchanged: []string{"versions.tf"}},
		{name: "declaration_earlier_file", files: map[string]string{"a.tf": old, "versions.tf": keep}, want: map[string]int{"a.tf": 1}, unchanged: []string{"versions.tf"}},
		{name: "all_constraints", files: map[string]string{"a.tf": old + old, "b.tofu": old, "versions.tf": "terraform {}\n"}, want: map[string]int{"a.tf": 2, "b.tofu": 1}, unchanged: []string{"versions.tf"}},
		{name: "nonliteral_counts", files: map[string]string{"main.tf": "terraform { required_version = var.runtime }\n"}, want: map[string]int{"main.tf": 1}, absent: []string{"versions.tf"}},
		{name: "null_counts", files: map[string]string{"main.tf": "terraform { required_version = null }\n"}, want: map[string]int{"main.tf": 1}, absent: []string{"versions.tf"}},
		{name: "empty_string_counts", files: map[string]string{"main.tf": "terraform { required_version = \"\" }\n"}, want: map[string]int{"main.tf": 1}, absent: []string{"versions.tf"}},
		{name: "labeled_nested_root_decoys", files: map[string]string{"main.tf": "required_version = \"root\"\nterraform \"label\" { required_version = \"label\" }\nlocals {\n terraform { required_version = \"nested\" }\n}\n# terraform { required_version = \"comment\" }\n"}, unchanged: []string{"main.tf"}, want: map[string]int{"versions.tf": 1}},
		{name: "destination_decoys", files: map[string]string{"versions.tf": "terraform \"label\" { required_version = \"old\" }\nlocals {\n terraform { required_version = \"nested\" }\n}\n"}, want: map[string]int{"versions.tf": 1}, snippets: map[string][]string{"versions.tf": {"\"old\"", "\"nested\""}}},
		{name: "explicit_empty", path: "custom.tf", files: map[string]string{"custom.tf": ""}, want: map[string]int{"custom.tf": 1}, absent: []string{"versions.tf"}},
		{name: "explicit_unrelated", path: "custom.tofu", files: map[string]string{"custom.tofu": keep}, want: map[string]int{"custom.tofu": 1}, absent: []string{"versions.tf"}},
		{name: "explicit_hcl", path: "custom.hcl", files: map[string]string{"custom.hcl": keep}, want: map[string]int{"custom.hcl": 1}, absent: []string{"versions.tf"}},
		{name: "explicit_existing_hcl_constraint", path: "custom.hcl", files: map[string]string{"custom.hcl": old, "other.tf": old}, want: map[string]int{"custom.hcl": 1}, unchanged: []string{"other.tf"}, absent: []string{"versions.tf"}},
		{name: "explicit_sibling_suppresses", path: "a.tf", files: map[string]string{"a.tf": keep, "z.tofu": old}, unchanged: []string{"a.tf", "z.tofu"}, absent: []string{"versions.tf"}},
		{name: "explicit_both_have_constraints", path: "z.tf", files: map[string]string{"z.tf": old + old, "a.tofu": old}, want: map[string]int{"z.tf": 2}, unchanged: []string{"a.tofu"}, absent: []string{"versions.tf"}},
		{name: "missing_tf", path: "new.tf", want: map[string]int{"new.tf": 1}, absent: []string{"versions.tf"}},
		{name: "missing_tofu", path: "new.tofu", want: map[string]int{"new.tofu": 1}, absent: []string{"versions.tf"}},
		{name: "missing_with_sibling", path: "new.tofu", files: map[string]string{"a.tf": old}, unchanged: []string{"a.tf"}, absent: []string{"new.tofu", "versions.tf"}},
		{name: "missing_with_empty_sibling_value", path: "new.tf", files: map[string]string{"a.tofu": "terraform { required_version = \"\" }\n"}, unchanged: []string{"a.tofu"}, absent: []string{"new.tf", "versions.tf"}},
		{name: "explicit_empty_with_expression_sibling", path: "new.tf", files: map[string]string{"new.tf": "", "other.tf": "terraform { required_version = var.version }\n"}, unchanged: []string{"new.tf", "other.tf"}},
		{name: "ignore_sibling", path: "new.tf", flags: []string{"-i", "other"}, files: map[string]string{"other.tofu": old}, want: map[string]int{"new.tf": 1}, unchanged: []string{"other.tofu"}},
		{name: "ignore_existing_target", path: "target.tf", flags: []string{"-i", "target"}, files: map[string]string{"target.tf": keep}, unchanged: []string{"target.tf"}, absent: []string{"versions.tf"}},
		{name: "ignore_missing_target", path: "target.tf", flags: []string{"-i", "target"}, absent: []string{"target.tf", "versions.tf"}},
		{name: "ignore_destination", flags: []string{"-i", "versions"}, files: map[string]string{"main.tf": keep}, unchanged: []string{"main.tf"}, absent: []string{"versions.tf"}},
		{name: "ignore_existing_destination", flags: []string{"-i", "versions"}, files: map[string]string{"versions.tf": "terraform {}\n", "main.tf": keep}, unchanged: []string{"versions.tf", "main.tf"}},
		{name: "ignored_declaration_does_not_count", flags: []string{"-i", "other"}, files: map[string]string{"other.tf": old}, unchanged: []string{"other.tf"}, want: map[string]int{"versions.tf": 1}},
		{name: "ignored_destination_update_elsewhere", flags: []string{"-i", "versions"}, files: map[string]string{"other.tf": old, "versions.tf": keep}, unchanged: []string{"versions.tf"}, want: map[string]int{"other.tf": 1}},
		{name: "ignore_multiple", flags: []string{"-i", "a\\.tf$", "-i", "b\\.tofu$"}, files: map[string]string{"a.tf": old, "b.tofu": old}, unchanged: []string{"a.tf", "b.tofu"}, want: map[string]int{"versions.tf": 1}},
		{name: "nonrecursive", files: map[string]string{"child/main.tf": old}, want: map[string]int{"versions.tf": 1}, unchanged: []string{"child/main.tf"}, absent: []string{"child/versions.tf"}},
		{name: "recursive_mixed", flags: []string{"-r", "-i", "ignored"}, files: map[string]string{"main.tf": old, "child/main.tofu": keep, "constrained/a.tf": old, ".hidden/main.tf": old, "ignored/main.tf": old}, dirs: []string{"empty"}, want: map[string]int{"main.tf": 1, "child/versions.tf": 1, "constrained/a.tf": 1, "empty/versions.tf": 1}, unchanged: []string{"child/main.tofu", ".hidden/main.tf", "ignored/main.tf"}, absent: []string{"versions.tf", "constrained/versions.tf", ".hidden/versions.tf", "ignored/versions.tf"}},
		{name: "recursive_children_do_not_suppress_parent", flags: []string{"-r"}, files: map[string]string{"child/main.tf": old}, want: map[string]int{"versions.tf": 1, "child/main.tf": 1}, absent: []string{"child/versions.tf"}},
		{name: "recursive_child_ignored_destination", flags: []string{"-r", "-i", "child/versions"}, dirs: []string{"child", "sibling"}, want: map[string]int{"versions.tf": 1, "sibling/versions.tf": 1}, absent: []string{"child/versions.tf"}},
		{name: "lockfile_does_not_count", files: map[string]string{".terraform.lock.hcl": old}, want: map[string]int{"versions.tf": 1}, unchanged: []string{".terraform.lock.hcl"}},
		{name: "malformed_lockfile_ignored", files: map[string]string{".terraform.lock.hcl": "not { valid"}, want: map[string]int{"versions.tf": 1}, unchanged: []string{".terraform.lock.hcl"}},
		{name: "explicit_lockfile", path: ".terraform.lock.hcl", files: map[string]string{".terraform.lock.hcl": old}, unchanged: []string{".terraform.lock.hcl"}, absent: []string{"versions.tf"}},
		{name: "ignored_malformed_sibling", path: "new.tf", flags: []string{"-i", "broken"}, files: map[string]string{"broken.tf": "terraform {"}, unchanged: []string{"broken.tf"}, want: map[string]int{"new.tf": 1}},
		{name: "literal_special", version: "${var.example} %{ if true } quote\" slash\\ line\nnext", want: map[string]int{"versions.tf": 1}},
		{name: "literal_update", version: "${var.example} %{ if true }", files: map[string]string{"main.tf": old}, want: map[string]int{"main.tf": 1}, absent: []string{"versions.tf"}},
		{name: "constraint_range", version: ">= 1.4, < 2.0", want: map[string]int{"versions.tf": 1}},
		{name: "same_value_bytes", version: "old", files: map[string]string{"main.tf": "terraform { required_version = \"old\" }\n", "other.tf": keep}, want: map[string]int{"main.tf": 1}, unchanged: []string{"main.tf", "other.tf"}, absent: []string{"versions.tf"}},
	}
	for _, command := range []string{"terraform", "opentofu"} {
		for _, absolute := range []bool{false, true} {
			for _, tc := range cases {
				t.Run(fmt.Sprintf("%s/absolute_%t/%s", command, absolute, tc.name), func(t *testing.T) {
					// Absence-only checks must also demonstrate that the feature is available.
					if len(tc.want) == 0 {
						requireAddMissing(t, bin, command)
					}
					dir := t.TempDir()
					for name, content := range tc.files {
						writeVersionFixture(t, dir, name, content)
					}
					for _, name := range tc.dirs {
						if err := os.MkdirAll(filepath.Join(dir, name), 0755); err != nil {
							t.Fatal(err)
						}
					}
					path := tc.path
					if path == "" {
						path = "."
					}
					if absolute {
						path = filepath.Join(dir, path)
					}
					version := tc.version
					if version == "" {
						version = "new-runtime"
					}
					args := append([]string{command, "--add-missing", "-v", version}, tc.flags...)
					args = append(args, path)
					output, err := runVersionCLI(bin, dir, args...)
					if err != nil {
						t.Fatalf("CLI: %v\n%s", err, output)
					}
					for name, count := range tc.want {
						values, blocks := versionValues(t, readVersionFixture(t, dir, name), name)
						if len(values) != count {
							t.Fatalf("%s: constraints = %q, want %d", name, values, count)
						}
						for _, got := range values {
							if got != version {
								t.Errorf("%s: version %q, want %q", name, got, version)
							}
						}
						if want, ok := tc.blocks[name]; ok && blocks != want {
							t.Errorf("%s: %d blocks, want %d", name, blocks, want)
						}
					}
					if tc.name == "reuse_first_block" {
						src := readVersionFixture(t, dir, "versions.tf")
						f, d := hclsyntax.ParseConfig([]byte(src), "versions.tf", hcl.InitialPos)
						if d.HasErrors() {
							t.Fatal(d)
						}
						blocks := f.Body.(*hclsyntax.Body).Blocks
						if blocks[0].Body.Attributes["required_version"] == nil || blocks[1].Body.Attributes["required_version"] != nil {
							t.Fatal("must insert only into first eligible block")
						}
					}
					for _, name := range tc.unchanged {
						if got := readVersionFixture(t, dir, name); got != tc.files[name] {
							t.Errorf("%s changed unexpectedly: %q", name, got)
						}
					}
					for _, name := range tc.absent {
						if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
							t.Errorf("%s should remain absent, stat: %v", name, err)
						}
					}
					for name, snippets := range tc.snippets {
						for _, snippet := range snippets {
							if !strings.Contains(readVersionFixture(t, dir, name), snippet) {
								t.Errorf("%s lost %q", name, snippet)
							}
						}
					}
					before := versionTree(t, dir)
					output, err = runVersionCLI(bin, dir, args...)
					if err != nil {
						t.Fatalf("rerun: %v\n%s", err, output)
					}
					after := versionTree(t, dir)
					if fmt.Sprint(before) != fmt.Sprint(after) {
						t.Errorf("identical rerun changed files:\nbefore=%q\nafter=%q", before, after)
					}
					// No unrequested destination may appear anywhere in the fixture tree.
					expected := map[string]bool{}
					for name := range tc.files {
						expected[filepath.Clean(name)] = true
					}
					for name := range tc.want {
						expected[filepath.Clean(name)] = true
					}
					for name := range after {
						if !expected[name] {
							t.Errorf("unexpected file: %s", name)
						}
					}
				})
			}
		}
	}
}

func versionTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		contents, err := root.ReadFile(rel)
		if err != nil {
			return err
		}
		files[rel] = string(contents)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestAddMissingErrors(t *testing.T) {
	bin := buildVersionCLI(t)
	cases := []struct {
		name, path  string
		files       map[string]string
		dirs, flags []string
		permission  string
		writeError  bool
	}{
		{name: "missing_parent", path: "absent/new.tf"},
		{name: "missing_hcl", path: "new.hcl"},
		{name: "missing_json", path: "new.tf.json"},
		{name: "missing_no_extension", path: "new"},
		{name: "missing_lockfile", path: ".terraform.lock.hcl"},
		{name: "syntax_same_module", files: map[string]string{"a.tf": "terraform { required_version = \"old\" }\n", "z.tf": "terraform {"}},
		{name: "syntax_late_descendant", flags: []string{"-r"}, files: map[string]string{"a/main.tf": "terraform { required_version = \"old\" }\n", "z/main.tf": "terraform {"}},
		{name: "syntax_sibling_existing", path: "target.tf", files: map[string]string{"target.tf": "terraform { required_version = \"old\" }\n", "z.tofu": "terraform {"}},
		{name: "syntax_sibling_missing", path: "target.tofu", files: map[string]string{"z.tf": "terraform {"}},
		{name: "syntax_explicit_hcl", path: "target.hcl", files: map[string]string{"target.hcl": "terraform {"}},
		{name: "destination_is_directory", dirs: []string{"versions.tf"}},
		{name: "recursive_destination_conflict_root_creation", flags: []string{"-r"}, files: map[string]string{"main.tf": "locals { value = 1 }\n"}, dirs: []string{"child/versions.tf"}},
		{name: "recursive_destination_conflict_root_update", flags: []string{"-r"}, files: map[string]string{"main.tf": "terraform { required_version = \"old\" }\n"}, dirs: []string{"child/versions.tf"}},
		{name: "recursive_destination_conflict_later_sibling", flags: []string{"-r"}, files: map[string]string{"a/main.tofu": "locals { value = 1 }\n"}, dirs: []string{"z/versions.tf"}},
		{name: "read_failure", files: map[string]string{"a.tf": "terraform { required_version = \"old\" }\n", "z.tf": ""}, permission: "z.tf"},
		{name: "discovery_late_descendant", flags: []string{"-r"}, files: map[string]string{"a/main.tf": "terraform { required_version = \"old\" }\n"}, dirs: []string{"z"}, permission: "z"},
		{name: "explicit_parent_not_directory", path: "parent/new.tf", files: map[string]string{"parent": ""}},
		{name: "explicit_stat_denied", path: "denied/new.tf", dirs: []string{"denied"}, permission: "denied"},
		{name: "explicit_hcl_read_failure", path: "target.hcl", files: map[string]string{"target.hcl": ""}, permission: "target.hcl"},
		{name: "destination_stat_denied", path: "module", dirs: []string{"module"}, permission: "module"},
		{name: "destination_write_denied", path: "module", dirs: []string{"module"}, permission: "module", writeError: true},
		{name: "write_failure", files: map[string]string{"main.tf": "terraform { required_version = \"old\" }\n"}, permission: "main.tf", writeError: true},
	}
	for _, command := range []string{"terraform", "opentofu"} {
		for _, tc := range cases {
			t.Run(command+"/"+tc.name, func(t *testing.T) {
				requireAddMissing(t, bin, command)
				if tc.permission != "" && os.Geteuid() == 0 {
					t.Skip("permission cases require non-root execution")
				}
				dir := t.TempDir()
				for name, src := range tc.files {
					writeVersionFixture(t, dir, name, src)
				}
				for _, name := range tc.dirs {
					if err := os.MkdirAll(filepath.Join(dir, name), 0755); err != nil {
						t.Fatal(err)
					}
				}
				before := versionTree(t, dir)
				if tc.permission != "" {
					path := filepath.Join(dir, tc.permission)
					mode := os.FileMode(0)
					if tc.writeError {
						mode = 0444
						if tc.name == "destination_write_denied" {
							mode = 0555
						}
					}
					if tc.name == "discovery_late_descendant" {
						mode = 0111
					}
					if tc.name == "destination_stat_denied" {
						mode = 0444
					}
					if err := os.Chmod(path, mode); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = os.Chmod(path, 0755) })
				}
				path := tc.path
				if path == "" {
					path = "."
				}
				args := append([]string{command, "--add-missing", "-v", "new-runtime"}, tc.flags...)
				args = append(args, path)
				output, err := runVersionCLI(bin, dir, args...)
				if err == nil {
					t.Fatalf("expected error, output: %s", output)
				}
				if len(strings.TrimSpace(string(output))) == 0 {
					t.Error("error must be reported")
				}
				if tc.permission != "" {
					if err := os.Chmod(filepath.Join(dir, tc.permission), 0755); err != nil {
						t.Fatal(err)
					}
				}
				if !tc.writeError {
					after := versionTree(t, dir)
					if fmt.Sprint(before) != fmt.Sprint(after) {
						t.Errorf("preflight failure changed files: before=%q after=%q", before, after)
					}
				}
				if tc.name == "missing_parent" {
					if _, err := os.Stat(filepath.Join(dir, "absent")); !os.IsNotExist(err) {
						t.Error("created parent directory")
					}
				}
			})
		}
	}
}

func TestAddMissingDocumentation(t *testing.T) {
	bin := buildVersionCLI(t)
	for _, command := range []string{"terraform", "opentofu"} {
		t.Run(command, func(t *testing.T) {
			out, _ := runVersionCLI(bin, t.TempDir(), command, "--help")
			if !strings.Contains(string(out), "--add-missing") {
				t.Fatalf("help must document flag: %s", out)
			}
		})
	}
}

func TestAddMissingCompatibility(t *testing.T) {
	bin := buildVersionCLI(t)
	for _, command := range []string{"terraform", "opentofu"} {
		t.Run(command, func(t *testing.T) {
			requireAddMissing(t, bin, command)
			dir := t.TempDir()
			writeVersionFixture(t, dir, "main.tf", "locals { value=1 }\n")
			before := versionTree(t, dir)
			out, err := runVersionCLI(bin, dir, command, "-v", "new-runtime", ".")
			if err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			if fmt.Sprint(before) != fmt.Sprint(versionTree(t, dir)) {
				t.Fatal("default command created a constraint")
			}
		})
	}
	for _, command := range []string{"provider", "module", "lock"} {
		t.Run(command, func(t *testing.T) {
			requireAddMissing(t, bin, "terraform")
			dir := t.TempDir()
			out, err := runVersionCLI(bin, dir, command, "--add-missing", ".")
			if err == nil {
				t.Fatalf("unsupported flag accepted: %s", out)
			}
			if len(versionTree(t, dir)) != 0 {
				t.Fatal("unsupported command created files")
			}
		})
	}
}

func TestAddMissingDefaultRegressions(t *testing.T) {
	bin := buildVersionCLI(t)
	for _, command := range []string{"terraform", "opentofu"} {
		for _, disabled := range []bool{false, true} {
			for _, recursive := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/disabled_%t/relative_ignore_recursive_%t", command, disabled, recursive), func(t *testing.T) {
					requireAddMissing(t, bin, command)
					dir := filepath.Join(t.TempDir(), "ignore-prefix-collision")
					old := "terraform { required_version = \"old\" }\n"
					writeVersionFixture(t, dir, "module/main.tf", old)
					writeVersionFixture(t, dir, "module/child/main.tf", old)
					args := []string{command, "-v", "new-runtime", "-i", "ignore-prefix-collision"}
					if disabled {
						args = append(args, "--add-missing=false")
					}
					if recursive {
						args = append(args, "-r")
					}
					args = append(args, "./module")
					output, err := runVersionCLI(bin, dir, args...)
					if err != nil {
						t.Fatalf("default update: %v\n%s", err, output)
					}
					values, _ := versionValues(t, readVersionFixture(t, dir, "module/main.tf"), "main.tf")
					if len(values) != 1 || values[0] != "new-runtime" {
						t.Fatalf("default relative ignore changed behavior: got %q", values)
					}
					child := readVersionFixture(t, dir, "module/child/main.tf")
					if recursive {
						values, _ := versionValues(t, child, "child/main.tf")
						if len(values) != 1 || values[0] != "new-runtime" {
							t.Fatalf("default recursive relative ignore changed behavior: got %q", values)
						}
					} else if child != old {
						t.Fatal("nonrecursive default update changed child")
					}
				})
			}
			for _, target := range []string{"directory", "file", "recursive"} {
				t.Run(fmt.Sprintf("%s/disabled_%t/malformed_lock_%s", command, disabled, target), func(t *testing.T) {
					requireAddMissing(t, bin, command)
					dir := t.TempDir()
					name := "module/.terraform.lock.hcl"
					path := "module"
					args := []string{command, "-v", "new-runtime"}
					if disabled {
						args = append(args, "--add-missing=false")
					}
					if target == "recursive" {
						name = "module/child/.terraform.lock.hcl"
						args = append(args, "-r")
					}
					if target == "file" {
						path = name
					}
					malformed := "provider \"registry.example/test/provider\" {\n"
					writeVersionFixture(t, dir, name, malformed)
					output, err := runVersionCLI(bin, dir, append(args, path)...)
					if err == nil {
						t.Fatal("default command silently accepted malformed lockfile")
					}
					if strings.TrimSpace(string(output)) == "" {
						t.Fatal("default parse error was not reported")
					}
					if got := readVersionFixture(t, dir, name); got != malformed {
						t.Fatal("malformed lockfile changed")
					}
				})
			}
		}
	}
}
