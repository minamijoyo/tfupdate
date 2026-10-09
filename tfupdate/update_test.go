package tfupdate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"runtime/debug"
	"strings"
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
				updateType: "opentofu",
				version:    "1.9.0",
			},
			want: &OpenTofuUpdater{
				version: "1.9.0",
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
			cmp.AllowUnexported(OpenTofuUpdater{}),
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

// runJSON exercises the existing public stream interface with real module context.
func runJSON(t *testing.T, o Option, input string, neighbors map[string]string) (string, bool, error) {
	t.Helper()
	fs := afero.NewMemMapFs()
	for name, content := range neighbors {
		if err := afero.WriteFile(fs, name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := afero.WriteFile(fs, "case.tf.json", []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
	gc, err := NewGlobalContext(fs, o)
	if err != nil {
		t.Fatal(err)
	}
	mc, err := NewModuleContext(".", gc)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	changed, err := UpdateHCL(context.Background(), mc, strings.NewReader(input), &out, "case.tf.json")
	return out.String(), changed, err
}

func TestJSONUpdates(t *testing.T) {
	cases := []struct {
		name, kind, match, input, want string
		regex                          bool
		neighbors                      map[string]string
	}{
		{name: "terraform", kind: "terraform", input: `{"terraform":{"required_version":"old"}}`, want: `{"terraform":{"required_version":"2.4.0"}}`},
		{name: "opentofu", kind: "opentofu", input: `{"terraform":{"required_version":"old"}}`, want: `{"terraform":{"required_version":"2.4.0"}}`},
		{name: "empty_version", kind: "terraform", input: `{"terraform":{"required_version":""}}`, want: `{"terraform":{"required_version":"2.4.0"}}`},
		{name: "repeated_blocks", kind: "terraform", input: `{"terraform":{"required_version":"a"},"terraform":[{"required_version":"longer"},{"backend":{"s3":{}}},{"required_version":"b"}]}`, want: `{"terraform":{"required_version":"2.4.0"},"terraform":[{"required_version":"2.4.0"},{"backend":{"s3":{}}},{"required_version":"2.4.0"}]}`},
		{name: "decoded_keys_and_unrelated_values", kind: "terraform", input: `{"terraform":{"required_\u0076ersion":"old","//":"old"},"locals":{"required_version":"old","n":9007199254740993,"e":1.2300e+09,"s":"\u0061","expr":"${var.missing}","repeat":1,"repeat":2}}`, want: `{"terraform":{"required_\u0076ersion":"2.4.0","//":"old"},"locals":{"required_version":"old","n":9007199254740993,"e":1.2300e+09,"s":"\u0061","expr":"${var.missing}","repeat":1,"repeat":2}}`},
		{name: "null_then_valid", kind: "terraform", input: `{"terraform":null,"terraform":[],"terraform":{},"terraform":{"required_version":"old"}}`, want: `{"terraform":null,"terraform":[],"terraform":{},"terraform":{"required_version":"2.4.0"}}`},
		{name: "legacy_provider", kind: "provider", match: "aws", input: `{"terraform":{"required_providers":{"aws":"old","other":42}}}`, want: `{"terraform":{"required_providers":{"aws":"2.4.0","other":42}}}`},
		{name: "modern_provider", kind: "provider", match: "aws", input: `{"terraform":{"required_providers":{"aws":{"source":"hashicorp/aws","version":"old","configuration_aliases":["${aws.other}"]}}}}`, want: `{"terraform":{"required_providers":{"aws":{"source":"hashicorp/aws","version":"2.4.0","configuration_aliases":["${aws.other}"]}}}}`},
		{name: "empty_modern_version", kind: "provider", match: "aws", input: `{"terraform":{"required_providers":{"aws":{"version":""}}}}`, want: `{"terraform":{"required_providers":{"aws":{"version":"2.4.0"}}}}`},
		{name: "provider_forms_and_reverse_edit_order", kind: "provider", match: "aws", input: `{"provider":{"aws":[{"version":"a","alias":"east"},{"version":"longold"}],"other":{"version":3}},"terraform":[{"required_providers":[{"aws":"legacy"},{"aws":{"version":"modern"}}]}]}`, want: `{"provider":{"aws":[{"version":"2.4.0","alias":"east"},{"version":"2.4.0"}],"other":{"version":3}},"terraform":[{"required_providers":[{"aws":"2.4.0"},{"aws":{"version":"2.4.0"}}]}]}`},
		{name: "provider_missing_and_selected", kind: "provider", match: "aws", input: `{"terraform":[{"required_providers":{"aws":{"source":"hashicorp/aws"}}},{"required_providers":{"other":"old"}}],"provider":[{"aws":{}},{"aws":{"version":"old"}}]}`, want: `{"terraform":[{"required_providers":{"aws":{"source":"hashicorp/aws"}}},{"required_providers":{"other":"old"}}],"provider":[{"aws":{}},{"aws":{"version":"2.4.0"}}]}`},
		{name: "provider_null_collections", kind: "provider", match: "aws", input: `{"terraform":{"required_providers":null},"provider":null,"provider":[{"unused":null},{"aws":[]},{"aws":{"version":"old"}}]}`, want: `{"terraform":{"required_providers":null},"provider":null,"provider":[{"unused":null},{"aws":[]},{"aws":{"version":"2.4.0"}}]}`},
		{name: "qualified_provider_local_identity", kind: "provider", match: "vendor/cloud", input: `{"terraform":{"required_providers":{"local":{"source":"vendor/cloud","version":"old"},"cloud":{"source":"other/cloud","version":"leave"}}},"provider":{"local":{"version":"leave"}}}`, want: `{"terraform":{"required_providers":{"local":{"source":"vendor/cloud","version":"2.4.0"},"cloud":{"source":"other/cloud","version":"leave"}}},"provider":{"local":{"version":"leave"}}}`},
		{name: "neighbor_native_identity", kind: "provider", match: "vendor/cloud", input: `{"terraform":{"required_providers":{"local":{"version":"old"}}}}`, want: `{"terraform":{"required_providers":{"local":{"version":"2.4.0"}}}}`, neighbors: map[string]string{"providers.tf": "terraform {\n required_providers {\n local = { source = \"vendor/cloud\" }\n }\n}\n"}},
		{name: "neighbor_json_identity", kind: "provider", match: "vendor/cloud", input: `{"terraform":{"required_providers":{"local":{"version":"old"}}}}`, want: `{"terraform":{"required_providers":{"local":{"version":"2.4.0"}}}}`, neighbors: map[string]string{"providers.tf.json": `{"terraform":{"required_providers":{"local":{"source":"vendor/cloud"}}}}`}},
		{name: "module_registry", kind: "module", match: "vendor/mod/cloud", input: `{"module":{"different_label":{"source":"vendor/mod/cloud","version":"old","providers":{"x":"${x.y}"}},"other":{"source":"unmatched","version":false}}}`, want: `{"module":{"different_label":{"source":"vendor/mod/cloud","version":"2.4.0","providers":{"x":"${x.y}"}},"other":{"source":"unmatched","version":false}}}`},
		{name: "module_ref_precedence", kind: "module", match: "git::https://example.test/mod", input: `{"module":{"x":{"source":"git::https://example.test/mod?ref=v1.2.3-rc.1","version":"leave"}}}`, want: `{"module":{"x":{"source":"git::https://example.test/mod?ref=v2.4.0","version":"leave"}}}`},
		{name: "module_regex", kind: "module", match: `^git::https://example[.]test/(one|two)$`, regex: true, input: `{"module":[{"x":{"source":"git::https://example.test/one?ref=v1","version":"leave"}},{"y":{"source":"git::https://example.test/two","version":"old"}},{"z":{"source":"git::https://example.test/three","version":null}}]}`, want: `{"module":[{"x":{"source":"git::https://example.test/one?ref=v2.4.0","version":"leave"}},{"y":{"source":"git::https://example.test/two","version":"2.4.0"}},{"z":{"source":"git::https://example.test/three","version":null}}]}`},
		{name: "module_literal_decoded_source", kind: "module", match: `${var.repo}/mod`, input: `{"module":{"x":{"source":"${var.repo}\/m\u006fd","version":"${literal}"}}}`, want: `{"module":{"x":{"source":"${var.repo}\/m\u006fd","version":"2.4.0"}}}`},
		{name: "module_no_source_or_version", kind: "module", match: "target", input: `{"module":{"missing_source":{"version":null},"missing_version":{"source":"target"},"matched":{"source":"target","version":"old"}}}`, want: `{"module":{"missing_source":{"version":null},"missing_version":{"source":"target"},"matched":{"source":"target","version":"2.4.0"}}}`},
		{name: "module_branch_reference_literal", kind: "module", match: "target?ref=main", input: `{"module":{"x":{"source":"target?ref=main","version":"old"},"y":{"source":"prefix-target?ref=main","version":"leave"}}}`, want: `{"module":{"x":{"source":"target?ref=main","version":"2.4.0"},"y":{"source":"prefix-target?ref=main","version":"leave"}}}`},
		{name: "unrelated_block_shape", kind: "terraform", input: `{"module":false,"provider":9,"terraform":{"required_version":"old"}}`, want: `{"module":false,"provider":9,"terraform":{"required_version":"2.4.0"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := Option{updateType: tc.kind, name: tc.match, version: "2.4.0"}
			if tc.regex {
				o.nameRegex = regexp.MustCompile(tc.match)
			}
			got, changed, err := runJSON(t, o, tc.input, tc.neighbors)
			if err != nil || !changed || got != tc.want {
				t.Errorf("first update: changed=%v err=%v\ngot  %s\nwant %s", changed, err, got, tc.want)
			}
			got, changed, err = runJSON(t, o, tc.want, tc.neighbors)
			if err != nil || changed || got != tc.want {
				t.Errorf("identical update: changed=%v err=%v\ngot %s\nwant %s", changed, err, got, tc.want)
			}
		})
	}
}

func TestJSONGeometry(t *testing.T) {
	for _, kind := range []string{"provider", "module"} {
		for _, labelsArray := range []bool{false, true} {
			for _, bodiesArray := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s_labels_%v_bodies_%v", kind, labelsArray, bodiesArray), func(t *testing.T) {
					body := `{"version":"before"}`
					if kind == "module" {
						body = `{"source":"wanted","version":"before"}`
					}
					if bodiesArray {
						body = "[" + body + "," + body + "]"
					}
					labels := `{"wanted":` + body + `}`
					if labelsArray {
						labels = "[" + labels + "," + labels + "]"
					}
					src := `{"` + kind + `":` + labels + `}`
					want := strings.ReplaceAll(src, `"before"`, `"after"`)
					got, changed, err := runJSON(t, Option{updateType: kind, name: "wanted", version: "after"}, src, nil)
					if err != nil || !changed || got != want {
						t.Errorf("changed=%v err=%v\ngot %s\nwant %s", changed, err, got, want)
					}
				})
			}
		}
	}
	for _, src := range []string{
		"{\r\n \"terraform\" : [ {\"required_version\" : \"x\"}, {\"required_version\":\"longlonglong\"} ]\r\n}",
		" {\t\"terraform\":{\"required_version\":\"x\"}, \"terraform\":{\"required_version\":\"longlonglong\"}} \n",
	} {
		t.Run(fmt.Sprintf("layout_%x", src[:2]), func(t *testing.T) {
			want := strings.ReplaceAll(strings.ReplaceAll(src, `"x"`, `"2"`), `"longlonglong"`, `"2"`)
			got, changed, err := runJSON(t, Option{updateType: "terraform", version: "2"}, src, nil)
			if err != nil || !changed || got != want {
				t.Errorf("changed=%v err=%v\ngot %q\nwant %q", changed, err, got, want)
			}
		})
	}
}

func TestJSONNoop(t *testing.T) {
	cases := []struct{ name, kind, match, input string }{
		{"empty", "terraform", "", `{}`},
		{"missing", "terraform", "", `{"terraform":{}}`},
		{"escaped_equal", "terraform", "", `{"terraform":{"required_version":"\u0032.4.0"}}`},
		{"null_blocks", "terraform", "", `{"terraform":null}`},
		{"empty_array", "terraform", "", `{"terraform":[]}`},
		{"unmatched_provider", "provider", "missing", `{"terraform":{"required_providers":{"other":{"version":false}}},"provider":{"other":{"version":null}}}`},
		{"unknown_source", "provider", "unknown/cloud", `{"terraform":{"required_providers":{"local":{"source":"vendor/cloud","version":"old"}}}}`},
		{"provider_absent", "provider", "aws", `{"terraform":[{}, {"required_providers":{}}]}`},
		{"module_empty", "module", "wanted", `{"module":[]}`},
		{"module_null_body", "module", "wanted", `{"module":{"x":null}}`},
		{"module_missing_version", "module", "wanted", `{"module":{"x":{"source":"wanted"}}}`},
		{"module_missing_source", "module", "wanted", `{"module":{"x":{"version":null}}}`},
		{"module_full_not_substring", "module", "wanted", `{"module":{"x":{"source":"prefix-wanted","version":"old"}}}`},
		{"module_branch_not_version", "module", "wanted", `{"module":{"x":{"source":"wanted?ref=main","version":"old"}}}`},
		{"lock", "lock", "", ` {"terraform":{"required_providers":{"x":{"source":"vendor/x","version":"old"}}}} `},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, err := runJSON(t, Option{updateType: tc.kind, name: tc.match, version: "2.4.0"}, tc.input, nil)
			if err != nil || changed || got != tc.input {
				t.Errorf("changed=%v err=%v\ngot %q\nwant %q", changed, err, got, tc.input)
			}
		})
	}
}

func TestJSONRejections(t *testing.T) {
	cases := []struct{ name, kind, input string }{
		{"empty", "terraform", ``}, {"truncated", "terraform", `{"terraform":`},
		{"trailing_document", "terraform", `{"terraform":{"required_version":"old"}} {}`},
		{"root_array", "terraform", `[]`}, {"root_null", "terraform", `null`}, {"root_scalar", "terraform", `"x"`},
		{"scalar_blocks", "terraform", `{"terraform":false}`},
		{"array_scalar", "terraform", `{"terraform":[{"required_version":"old"},4]}`},
		{"nested_arrays", "terraform", `{"terraform":[[{"required_version":"old"}]]}`},
		{"duplicate_version", "terraform", `{"terraform":{"required_version":"old","required_\u0076ersion":"other"}}`},
		{"duplicate_local", "provider", `{"terraform":{"required_providers":{"wanted":"old","wanted":"other"}}}`},
		{"duplicate_modern_version", "provider", `{"terraform":{"required_providers":{"wanted":{"version":"old","version":"other"}}}}`},
		{"duplicate_modern_source", "provider", `{"terraform":{"required_providers":{"wanted":{"source":"a/b","source":"c/d","version":"old"}}}}`},
		{"required_provider_scalar", "provider", `{"terraform":{"required_providers":false}}`},
		{"duplicate_unmatched_local", "provider", `{"terraform":{"required_providers":{"other":"a","other":"b"}}}`},
		{"duplicate_unmatched_provider_version", "provider", `{"provider":{"other":{"version":"a","version":"b"}}}`},
		{"duplicate_unmatched_modern_version", "provider", `{"terraform":{"required_providers":{"other":{"version":"a","version":"b"}}}}`},
		{"duplicate_unmatched_module_version", "module", `{"module":{"other":{"source":"unmatched","version":"a","version":"b"}}}`},
		{"duplicate_module_version_without_source", "module", `{"module":{"other":{"version":"a","version":"b"}}}`},
		{"provider_label_scalar", "provider", `{"provider":42}`},
		{"provider_body_scalar", "provider", `{"provider":{"wanted":42}}`},
		{"provider_body_array_scalar", "provider", `{"provider":{"wanted":[{"version":"old"},null]}}`},
		{"late_provider_bad_version", "provider", `{"provider":[{"wanted":{"version":"old"}},{"wanted":{"version":false}}]}`},
		{"module_bad_labels", "module", `{"module":[42]}`},
		{"module_bad_body", "module", `{"module":{"x":false}}`},
		{"duplicate_source", "module", `{"module":{"x":{"source":"wanted","source":"other","version":"old"}}}`},
		{"duplicate_module_version", "module", `{"module":{"x":{"source":"wanted","version":"old","version":"other"}}}`},
		{"late_bad_source", "module", `{"module":[{"x":{"source":"wanted","version":"old"}},{"y":{"source":null}}]}`},
	}
	for _, value := range []string{`null`, `false`, `9`, `{}`, `[]`} {
		cases = append(cases, struct{ name, kind, input string }{"runtime_value_" + value, "terraform", `{"terraform":[{"required_version":"old"},{"required_version":` + value + `}]}`})
		cases = append(cases, struct{ name, kind, input string }{"legacy_value_" + value, "provider", `{"provider":{"wanted":{"version":"old"}},"terraform":{"required_providers":{"wanted":` + value + `}}}`})
		// An object is a valid modern provider without a version.
		if value == `{}` {
			cases = cases[:len(cases)-1]
		}
		cases = append(cases, struct{ name, kind, input string }{"modern_value_" + value, "provider", `{"terraform":{"required_providers":{"wanted":{"version":` + value + `}}}}`})
		cases = append(cases, struct{ name, kind, input string }{"module_value_" + value, "module", `{"module":{"x":{"source":"wanted","version":` + value + `}}}`})
		cases = append(cases, struct{ name, kind, input string }{"source_value_" + value, "module", `{"module":{"x":{"source":` + value + `}}}`})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("JSON operation panicked: %v\n%s", r, debug.Stack())
				}
			}()
			// Check acceptance and rejection together: rejecting every JSON file is not a solution.
			good := `{"terraform":{"required_version":"old"}}`
			if tc.kind == "provider" {
				good = `{"provider":{"wanted":{"version":"old"}}}`
			}
			if tc.kind == "module" {
				good = `{"module":{"x":{"source":"wanted","version":"old"}}}`
			}
			o := Option{updateType: tc.kind, name: "wanted", version: "new"}
			got, changed, err := runJSON(t, o, good, nil)
			if err != nil || !changed || got != strings.ReplaceAll(good, `"old"`, `"new"`) {
				t.Errorf("valid counterpart: changed=%v err=%v output=%q", changed, err, got)
			}
			got, changed, err = runJSON(t, o, tc.input, nil)
			if err == nil || !strings.Contains(err.Error(), "case.tf.json") || changed || got != "" {
				t.Errorf("rejection: changed=%v err=%v output=%q", changed, err, got)
			}
			fs := afero.NewMemMapFs()
			if err := afero.WriteFile(fs, "case.tf.json", []byte(tc.input), 0644); err != nil {
				t.Fatal(err)
			}
			gc, err := NewGlobalContext(fs, o)
			if err != nil {
				t.Fatal(err)
			}
			if err := UpdateFileOrDir(context.Background(), gc, "case.tf.json"); err == nil {
				t.Error("invalid file accepted")
			}
			data, err := afero.ReadFile(fs, "case.tf.json")
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tc.input {
				t.Errorf("rejected file changed: %q", data)
			}
		})
	}
}

type jsonFailureReader struct{}

func (jsonFailureReader) Read(p []byte) (int, error) {
	return copy(p, `{"terraform":`), io.ErrUnexpectedEOF
}

type jsonFailureWriter struct{ short bool }

func (w jsonFailureWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) - 1, nil
	}
	return 0, io.ErrClosedPipe
}

func TestJSONIO(t *testing.T) {
	for _, mode := range []string{"read", "write", "short"} {
		t.Run(mode, func(t *testing.T) {
			o := Option{updateType: "terraform", version: "new"}
			src := `{"terraform":{"required_version":"old"}}`
			got, changed, err := runJSON(t, o, src, nil)
			if err != nil || !changed || got != `{"terraform":{"required_version":"new"}}` {
				t.Errorf("valid counterpart: %v %v %q", changed, err, got)
			}
			gc, err := NewGlobalContext(afero.NewMemMapFs(), o)
			if err != nil {
				t.Fatal(err)
			}
			mc, err := NewModuleContext(".", gc)
			if err != nil {
				t.Fatal(err)
			}
			var r io.Reader = strings.NewReader(src)
			var output bytes.Buffer
			var w io.Writer = &output
			if mode == "read" {
				r = jsonFailureReader{}
			} else {
				w = jsonFailureWriter{short: mode == "short"}
			}
			changed, err = UpdateHCL(context.Background(), mc, r, w, "io.tf.json")
			if err == nil || changed {
				t.Errorf("I/O failure lost: changed=%v err=%v", changed, err)
			}
			if mode == "read" && output.Len() != 0 {
				t.Errorf("output after read error: %q", output.String())
			}
		})
	}
}

func TestJSONLiteralReplacement(t *testing.T) {
	for _, value := range []string{"quote\"slash\\line\n", "\u03bb\U0001f642", "${literal}%{literal}", "a<b>&c"} {
		t.Run(fmt.Sprintf("value_%x", []byte(value)), func(t *testing.T) {
			src := "{\r\n \"terraform\":{\"required_version\":\"old\"}, \"locals\":{\"s\":\"\\u0061\",\"n\":1e9999}\r\n}"
			got, changed, err := runJSON(t, Option{updateType: "terraform", version: value}, src, nil)
			if err != nil || !changed {
				t.Fatalf("literal replacement: changed=%v err=%v", changed, err)
			}
			const prefix = "{\r\n \"terraform\":{\"required_version\":"
			const suffix = "}, \"locals\":{\"s\":\"\\u0061\",\"n\":1e9999}\r\n}"
			if !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) {
				t.Fatalf("surrounding bytes changed: %q", got)
			}
			var decoded string
			if err := json.Unmarshal([]byte(got[len(prefix):len(got)-len(suffix)]), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded != value {
				t.Errorf("decoded value=%q want=%q", decoded, value)
			}
			again, changed, err := runJSON(t, Option{updateType: "terraform", version: value}, got, nil)
			if err != nil || changed || again != got {
				t.Errorf("literal repeated update: changed=%v err=%v output=%q", changed, err, again)
			}
		})
	}
}

func TestJSONProviderValidationBeforeSelection(t *testing.T) {
	for _, field := range []string{"source", "version"} {
		t.Run(field, func(t *testing.T) {
			o := Option{updateType: "provider", name: "vendor/cloud", version: "new"}
			good := `{"terraform":{"required_providers":{"local":{"source":"vendor/cloud","version":"old"}}}}`
			got, changed, err := runJSON(t, o, good, nil)
			if err != nil || !changed || got != strings.ReplaceAll(good, `"old"`, `"new"`) {
				t.Errorf("valid qualified counterpart: %v %v %q", changed, err, got)
			}
			bad := strings.Replace(good, `"version":"old"`, `"version":"old","`+field+`":"duplicate"`, 1)
			got, changed, err = runJSON(t, o, bad, nil)
			if err == nil || !strings.Contains(err.Error(), "case.tf.json") || changed || got != "" {
				t.Errorf("duplicate hidden by metadata failure: %v %v %q", changed, err, got)
			}
		})
	}
	// The // property is a Terraform JSON comment in a block body, not a provider declaration.
	src := `{"terraform":{"required_providers":{"//":"one","//":"two","wanted":"old"}}}`
	want := strings.ReplaceAll(src, `"old"`, `"new"`)
	got, changed, err := runJSON(t, Option{updateType: "provider", name: "wanted", version: "new"}, src, nil)
	if err != nil || !changed || got != want {
		t.Errorf("comment properties: %v %v %q", changed, err, got)
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
			// We avoid this by formatting the output of this function.
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
terraform {
  required_version = "1.8.0"
}
`,
			o: Option{
				updateType: "opentofu",
				version:    "1.9.0",
			},
			want: `
terraform {
  required_version ="1.9.0"
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
