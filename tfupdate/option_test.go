package tfupdate

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/minamijoyo/tfupdate/tfregistry"
)

func TestNewOption(t *testing.T) {
	cases := []struct {
		updateType       string
		name             string
		version          string
		platforms        []string
		recursive        bool
		ignorePaths      []string
		sourceMatchType  string
		tfregistryConfig tfregistry.Config
		want             Option
		ok               bool
	}{
		{
			updateType:       "terraform",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "full",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "terraform",
				version:          "0.12.7",
				platforms:        []string{},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{},
				nameRegex:        nil,
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "provider",
			name:             "aws",
			version:          "2.23.0",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "full",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "provider",
				name:             "aws",
				version:          "2.23.0",
				platforms:        []string{},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{},
				nameRegex:        nil,
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "terraform",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{"hoge", "fuga"},
			sourceMatchType:  "full",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "terraform",
				version:          "0.12.7",
				platforms:        []string{},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{regexp.MustCompile("hoge"), regexp.MustCompile("fuga")},
				nameRegex:        nil,
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "terraform",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{""},
			sourceMatchType:  "full",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "terraform",
				version:          "0.12.7",
				platforms:        []string{},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{},
				nameRegex:        nil,
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "terraform",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{`\`},
			sourceMatchType:  "",
			tfregistryConfig: tfregistry.Config{},
			want:             Option{},
			ok:               false,
		},
		{
			updateType:       "lock",
			version:          "",
			platforms:        []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "full",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "lock",
				version:          "",
				platforms:        []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{},
				nameRegex:        nil,
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "module",
			name:             "terraform-aws-modules/vpc/aws",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "full",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "module",
				name:             "terraform-aws-modules/vpc/aws",
				version:          "0.12.7",
				platforms:        []string{},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{},
				nameRegex:        nil,
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "module",
			name:             `terraform-aws-modules\.git/vpc/aws`,
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "regex",
			tfregistryConfig: tfregistry.Config{},
			want: Option{
				updateType:       "module",
				name:             `terraform-aws-modules\.git/vpc/aws`,
				version:          "0.12.7",
				platforms:        []string{},
				recursive:        true,
				ignorePaths:      []*regexp.Regexp{},
				nameRegex:        regexp.MustCompile(`terraform-aws-modules\.git/vpc/aws`),
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
		{
			updateType:       "module",
			name:             "",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "regex",
			tfregistryConfig: tfregistry.Config{},
			ok:               false,
		},
		{
			updateType:       "module",
			name:             `\`,
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "regex",
			tfregistryConfig: tfregistry.Config{},
			ok:               false,
		},
		{
			updateType:       "module",
			name:             "",
			version:          "0.12.7",
			platforms:        []string{},
			recursive:        true,
			ignorePaths:      []string{},
			sourceMatchType:  "invalid",
			tfregistryConfig: tfregistry.Config{},
			ok:               false,
		},
	}

	for _, tc := range cases {
		got, err := NewOption(tc.updateType, tc.name, tc.version, tc.platforms, tc.recursive, tc.ignorePaths, tc.sourceMatchType, tc.tfregistryConfig)
		if tc.ok && err != nil {
			t.Errorf("NewOption() with updateType = %s, name = %s, version = %s, platforms = %#v, recursive = %t, ignorePath = %#v returns unexpected err: %+v", tc.updateType, tc.name, tc.version, tc.platforms, tc.recursive, tc.ignorePaths, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewOption() with updateType = %s, name = %s, version = %s, platforms = %#v, recursive = %t, ignorePath = %#v expects to return an error, but no error", tc.updateType, tc.name, tc.version, tc.platforms, tc.recursive, tc.ignorePaths)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewOption() with updateType = %s, name = %s, version = %s, platforms = %#v, recursive = %t, ignorePath = %#v returns %#v, but want = %#v", tc.updateType, tc.name, tc.version, tc.platforms, tc.recursive, tc.ignorePaths, got, tc.want)
		}
	}
}

func TestOptionMatchIgnorePaths(t *testing.T) {
	updateType := "terraform"
	version := "0.12.7"
	recursive := true

	cases := []struct {
		o    Option
		path string
		want bool
	}{
		{
			o: Option{
				updateType:  updateType,
				version:     version,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile(`.*\.tf`)},
			},
			path: "tmp/main.tf",
			want: true,
		},
		{
			o: Option{
				updateType:  updateType,
				version:     version,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile("tmp/"), regexp.MustCompile("hoge.tf")},
			},
			path: "tmp/main.tf",
			want: true,
		},
		{
			o: Option{
				updateType:  updateType,
				version:     version,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile("fuga/"), regexp.MustCompile("main.tf")},
			},
			path: "tmp/main.tf",
			want: true,
		},
		{
			o: Option{
				updateType:  updateType,
				version:     version,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile("fuga/"), regexp.MustCompile("test.tf")},
			},
			path: "tmp/main.tf",
			want: false,
		},
	}

	for _, tc := range cases {
		got := tc.o.MatchIgnorePaths(tc.path)
		if got != tc.want {
			t.Errorf("(*Option).MatchIgnorePaths() with option = %#v, path = %s returns %t, but want = %t", tc.o, tc.path, got, tc.want)
		}
	}
}
