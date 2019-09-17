package tfupdate

import (
	"testing"

	"github.com/spf13/afero"
)

func TestUpdateFile(t *testing.T) {
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
				target:     "0.12.7",
			},
			want: `
terraform {
  required_version = "0.12.7"
}
`,
			ok: true,
		},
		{
			filename: "invalid.tf",
			src: `
terraform {
  required_version = "0.12.6"
}
`,
			o: Option{
				updateType: "hoge",
				target:     "0.12.7",
			},
			want: `
terraform {
  required_version = "0.12.6"
}
`,
			ok: false,
		},
	}
	for _, tc := range cases {
		fs := afero.NewMemMapFs()
		afero.WriteFile(fs, tc.filename, []byte(tc.src), 0644)

		err := UpdateFile(fs, tc.filename, tc.o)
		if tc.ok && err != nil {
			t.Errorf("UpdateFile() with filename = %s, o = %#v returns unexpected err: %+v", tc.filename, tc.o, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("UpdateFile() with filename = %s, o = %#v expects to return an error, but no error: %+v", tc.filename, tc.o, err)
		}

		got, err := afero.ReadFile(fs, tc.filename)
		if err != nil {
			t.Fatalf("failed to read updated file: %s", err)
		}

		if string(got) != tc.want {
			t.Errorf("UpdateFile() with filename = %s, o = %#v returns %s, but want = %s", tc.filename, tc.o, got, tc.want)
		}
	}
}

func TestUpdateFileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	filename := "not_found.tf"
	o := Option{}

	err := UpdateFile(fs, filename, o)

	if err == nil {
		t.Errorf("UpdateFile() with filename = %s, o = %#v expects to return an error, but no error", filename, o)
	}
}
