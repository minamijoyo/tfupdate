package release

import (
	"reflect"
	"testing"
)

func TestSortVersions(t *testing.T) {
	cases := []struct {
		desc        string
		versionsRaw []string
		want        []string
	}{
		{
			desc:        "simple",
			versionsRaw: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1"},
			want:        []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0"},
		},
		{
			desc:        "empty",
			versionsRaw: []string{},
			want:        []string{},
		},
		{
			desc:        "pre-release",
			versionsRaw: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-alpha1", "0.3.0-rc"},
			want:        []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0-alpha1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-rc", "0.3.0"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := fromVersions(sortVersions(toVersions(tc.versionsRaw)))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got = %#v, but want = %#v", got, tc.want)
			}
		})
	}
}

func TestExcludePreReleases(t *testing.T) {
	cases := []struct {
		desc        string
		versionsRaw []string
		want        []string
	}{
		{
			desc:        "simple",
			versionsRaw: []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0-alpha1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-rc", "0.3.0"},
			want:        []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0"},
		},
		{
			desc:        "no pre-relases",
			versionsRaw: []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0"},
			want:        []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0"},
		},
		{
			desc:        "no stable relases",
			versionsRaw: []string{"0.3.0-alpha1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-rc"},
			want:        []string{},
		},
		{
			desc:        "empty",
			versionsRaw: []string{},
			want:        []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := fromVersions(excludePreReleases(toVersions(tc.versionsRaw)))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got = %#v, but want = %#v", got, tc.want)
			}
		})
	}
}
