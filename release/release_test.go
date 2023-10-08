package release

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockRelease struct {
	versions []string
	err      error
}

var _ Release = (*mockRelease)(nil)

func (r *mockRelease) ListReleases(ctx context.Context) ([]string, error) { // nolint revive unused-parameter
	return r.versions, r.err
}

func TestLatest(t *testing.T) {
	cases := []struct {
		desc string
		r    Release
		want string
		ok   bool
	}{
		{
			desc: "sort",
			r: &mockRelease{
				versions: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1"},
				err:      nil,
			},
			want: "0.3.0",
			ok:   true,
		},
		{
			desc: "pre-release",
			r: &mockRelease{
				versions: []string{"0.1.0", "0.2.0", "0.1.1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-alpha1", "0.3.0-rc"},
				err:      nil,
			},
			want: "0.2.0",
			ok:   true,
		},
		{
			desc: "no release",
			r: &mockRelease{
				versions: []string{},
				err:      nil,
			},
			want: "",
			ok:   false,
		},
		{
			desc: "api error",
			r: &mockRelease{
				versions: nil,
				err:      errors.New("mocked error"),
			},
			want: "",
			ok:   false,
		},
		{
			desc: "parse error",
			r: &mockRelease{
				versions: []string{"foo", "0.3.0", "0.2.0", "0.1.0", "0.1.1"},
				err:      nil,
			},
			want: "0.3.0",
			ok:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := Latest(context.Background(), tc.r)

			if tc.ok && err != nil {
				t.Fatalf("unexpected err: %#v", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expects to return an error, but no error. got = %#v", got)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got = %#v, but want = %#v", got, tc.want)
			}
		})
	}
}

func TestList(t *testing.T) {
	cases := []struct {
		desc       string
		r          Release
		maxLength  int
		preRelease bool
		want       []string
		ok         bool
	}{
		{
			desc: "sort",
			r: &mockRelease{
				versions: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1"},
				err:      nil,
			},
			maxLength:  5,
			preRelease: true,
			want:       []string{"0.1.0", "0.1.1", "0.2.0", "0.3.0"},
			ok:         true,
		},
		{
			desc: "maxLength",
			r: &mockRelease{
				versions: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1"},
				err:      nil,
			},
			maxLength:  3,
			preRelease: true,
			want:       []string{"0.1.1", "0.2.0", "0.3.0"},
			ok:         true,
		},
		{
			desc: "include pre-release",
			r: &mockRelease{
				versions: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-alpha1", "0.3.0-rc"},
				err:      nil,
			},
			maxLength:  3,
			preRelease: true,
			want:       []string{"0.3.0-beta2", "0.3.0-rc", "0.3.0"},
			ok:         true,
		},
		{
			desc: "exclude pre-release",
			r: &mockRelease{
				versions: []string{"0.3.0", "0.2.0", "0.1.0", "0.1.1", "0.3.0-beta1", "0.3.0-beta2", "0.3.0-alpha1", "0.3.0-rc"},
				err:      nil,
			},
			maxLength:  3,
			preRelease: false,
			want:       []string{"0.1.1", "0.2.0", "0.3.0"},
			ok:         true,
		},
		{
			desc: "empty",
			r: &mockRelease{
				versions: []string{},
				err:      nil,
			},
			maxLength:  3,
			preRelease: true,
			want:       []string{},
			ok:         true,
		},
		{
			desc: "api error",
			r: &mockRelease{
				versions: nil,
				err:      errors.New("mocked error"),
			},
			maxLength:  3,
			preRelease: true,
			want:       nil,
			ok:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := List(context.Background(), tc.r, tc.maxLength, tc.preRelease)

			if tc.ok && err != nil {
				t.Fatalf("unexpected err: %#v", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expects to return an error, but no error. got = %#v", got)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got = %#v, but want = %#v", got, tc.want)
			}
		})
	}
}
