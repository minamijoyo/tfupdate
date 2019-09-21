package tfupdate

import (
	"reflect"
	"regexp"
	"testing"
)

func TestNewOption(t *testing.T) {
	cases := []struct {
		updateType string
		target     string
		recursive  bool
		ignorePath string
		want       Option
		ok         bool
	}{
		{
			updateType: "terraform",
			target:     "0.12.7",
			recursive:  true,
			ignorePath: "hoge",
			want: Option{
				updateType: "terraform",
				target:     "0.12.7",
				recursive:  true,
				ignorePath: regexp.MustCompile("hoge"),
			},
			ok: true,
		},
		{
			updateType: "terraform",
			target:     "0.12.7",
			recursive:  true,
			ignorePath: "",
			want: Option{
				updateType: "terraform",
				target:     "0.12.7",
				recursive:  true,
				ignorePath: nil,
			},
			ok: true,
		},
		{
			updateType: "terraform",
			target:     "0.12.7",
			recursive:  true,
			ignorePath: `\`,
			want:       Option{},
			ok:         false,
		},
	}

	for _, tc := range cases {
		got, err := NewOption(tc.updateType, tc.target, tc.recursive, tc.ignorePath)
		if tc.ok && err != nil {
			t.Errorf("NewOption() with updateType = %s, target = %s, recursive = %t, ignorePath = %s returns unexpected err: %+v", tc.updateType, tc.target, tc.recursive, tc.ignorePath, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewOption() with updateType = %s, target = %s, recursive = %t, ignorePath = %s expects to return an error, but no error", tc.updateType, tc.target, tc.recursive, tc.ignorePath)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewOption() with updateType = %s, target = %s, recursive = %t, ignorePath = %s returns %#v, but want = %#v", tc.updateType, tc.target, tc.recursive, tc.ignorePath, got, tc.want)
		}
	}
}
