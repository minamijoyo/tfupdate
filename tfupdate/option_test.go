package tfupdate

import (
	"reflect"
	"regexp"
	"testing"
)

func TestNewOption(t *testing.T) {
	cases := []struct {
		updateType  string
		target      string
		recursive   bool
		ignorePaths []string
		want        Option
		ok          bool
	}{
		{
			updateType:  "terraform",
			target:      "0.12.7",
			recursive:   true,
			ignorePaths: []string{"hoge", "fuga"},
			want: Option{
				updateType:  "terraform",
				target:      "0.12.7",
				recursive:   true,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile("hoge"), regexp.MustCompile("fuga")},
			},
			ok: true,
		},
		{
			updateType:  "terraform",
			target:      "0.12.7",
			recursive:   true,
			ignorePaths: []string{""},
			want: Option{
				updateType:  "terraform",
				target:      "0.12.7",
				recursive:   true,
				ignorePaths: []*regexp.Regexp{},
			},
			ok: true,
		},
		{
			updateType:  "terraform",
			target:      "0.12.7",
			recursive:   true,
			ignorePaths: []string{`\`},
			want:        Option{},
			ok:          false,
		},
	}

	for _, tc := range cases {
		got, err := NewOption(tc.updateType, tc.target, tc.recursive, tc.ignorePaths)
		if tc.ok && err != nil {
			t.Errorf("NewOption() with updateType = %s, target = %s, recursive = %t, ignorePath = %#v returns unexpected err: %+v", tc.updateType, tc.target, tc.recursive, tc.ignorePaths, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewOption() with updateType = %s, target = %s, recursive = %t, ignorePath = %#v expects to return an error, but no error", tc.updateType, tc.target, tc.recursive, tc.ignorePaths)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewOption() with updateType = %s, target = %s, recursive = %t, ignorePath = %#v returns %#v, but want = %#v", tc.updateType, tc.target, tc.recursive, tc.ignorePaths, got, tc.want)
		}
	}
}

func TestOptionMatchIgnorePaths(t *testing.T) {
	updateType := "terraform"
	target := "0.12.7"
	recursive := true

	cases := []struct {
		o    Option
		path string
		want bool
	}{
		{
			o: Option{
				updateType:  updateType,
				target:      target,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile(`.*\.tf`)},
			},
			path: "tmp/main.tf",
			want: true,
		},
		{
			o: Option{
				updateType:  updateType,
				target:      target,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile("tmp/"), regexp.MustCompile("hoge.tf")},
			},
			path: "tmp/main.tf",
			want: true,
		},
		{
			o: Option{
				updateType:  updateType,
				target:      target,
				recursive:   recursive,
				ignorePaths: []*regexp.Regexp{regexp.MustCompile("fuga/"), regexp.MustCompile("main.tf")},
			},
			path: "tmp/main.tf",
			want: true,
		},
		{
			o: Option{
				updateType:  updateType,
				target:      target,
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
