package tfupdate

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestAllMatchingBlocks(t *testing.T) {
	src := `a = "b"
service {
  attr0 = "val0"
}
service {
  attr1 = "val1"
}
service "label1" "label2" {
  attr2 = "val2"
}
service "label1" "label2" {
  attr3 = "val3"
}
`

	tests := []struct {
		src      string
		typeName string
		labels   []string
		want     string
	}{
		{
			src,
			"service",
			[]string{},
			`service {
  attr0 = "val0"
}
service {
  attr1 = "val1"
}
`,
		},
		{
			src,
			"service",
			[]string{"label1", "label2"},
			`service "label1" "label2" {
  attr2 = "val2"
}
service "label1" "label2" {
  attr3 = "val3"
}
`,
		},
		{
			src,
			"hoge",
			[]string{},
			"",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.typeName, strings.Join(test.labels, " ")), func(t *testing.T) {
			f, diags := hclwrite.ParseConfig([]byte(test.src), "", hcl.Pos{Line: 1, Column: 1})
			if len(diags) != 0 {
				for _, diag := range diags {
					t.Logf("- %s", diag.Error())
				}
				t.Fatalf("unexpected diagnostics")
			}

			blocks := allMatchingBlocks(f.Body(), test.typeName, test.labels)
			if len(blocks) == 0 {
				if test.want != "" {
					t.Fatal("block not found, but want it to exist")
				}
			} else {
				if test.want == "" {
					t.Fatal("block found, but expecting not found")
				}

				got := ""
				for _, block := range blocks {
					got += string(block.BuildTokens(nil).Bytes())
				}
				if got != test.want {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.want)
				}
			}
		})
	}
}

func TestAllMatchingBlocksByType(t *testing.T) {
	src := `a = "b"
service {
  attr0 = "val0"
}
resource {
  attr1 = "val1"
}
service "label1" {
  attr2 = "val2"
}
`

	tests := []struct {
		src      string
		typeName string
		want     string
	}{
		{
			src,
			"service",
			`service {
  attr0 = "val0"
}
service "label1" {
  attr2 = "val2"
}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.typeName, func(t *testing.T) {
			f, diags := hclwrite.ParseConfig([]byte(test.src), "", hcl.Pos{Line: 1, Column: 1})
			if len(diags) != 0 {
				for _, diag := range diags {
					t.Logf("- %s", diag.Error())
				}
				t.Fatalf("unexpected diagnostics")
			}

			blocks := allMatchingBlocksByType(f.Body(), test.typeName)
			if len(blocks) == 0 {
				if test.want != "" {
					t.Fatal("block not found, but want it to exist")
				}
			} else {
				if test.want == "" {
					t.Fatal("block found, but expecting not found")
				}

				got := ""
				for _, block := range blocks {
					got += string(block.BuildTokens(nil).Bytes())
				}
				if got != test.want {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.want)
				}
			}
		})
	}
}

func TestGetHCLNativeAttributeValue(t *testing.T) {
	cases := []struct {
		desc         string
		src          string
		name         string
		wantExprType hcl.Expression
		ok           bool
	}{
		{
			desc: "string literal",
			src: `
foo = "123"
`,
			name:         "foo",
			wantExprType: &hclsyntax.TemplateExpr{},
			ok:           true,
		},
		{
			desc: "object literal",
			src: `
foo = {
  bar = "123"
  baz = "BAZ"
}
`,
			name:         "foo",
			wantExprType: &hclsyntax.ObjectConsExpr{},
			ok:           true,
		},
		{
			desc: "object with references",
			src: `
foo = {
  bar = "123"
  baz = "BAZ"

  items = [
    var.aaa,
    var.bbb,
  ]
}
`,
			name:         "foo",
			wantExprType: &hclsyntax.ObjectConsExpr{},
			ok:           true,
		},
		{
			desc: "not found",
			src: `
foo = "123"
`,
			name:         "bar",
			wantExprType: nil,
			ok:           true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			f, diags := hclwrite.ParseConfig([]byte(tc.src), "", hcl.Pos{Line: 1, Column: 1})
			if len(diags) != 0 {
				for _, diag := range diags {
					t.Logf("- %s", diag.Error())
				}
				t.Fatalf("unexpected diagnostics")
			}

			got, err := getHCLNativeAttribute(f.Body(), tc.name)
			if tc.ok && err != nil {
				t.Errorf("unexpected err: %#v", err)
			}

			if !tc.ok && err == nil {
				t.Errorf("expects to return an error, but no error. got = %#v", got)
			}

			if tc.ok && got != nil {
				// An expression is a complicated object and hard to build from literal.
				// So we simply compare it by type.
				if reflect.TypeOf(got.Expr) != reflect.TypeOf(tc.wantExprType) {
					t.Errorf("got = %#v, but want = %#v", got.Expr, tc.wantExprType)
				}
			}
		})
	}
}

func TestGetAttributeValueAsUnquotedString(t *testing.T) {
	cases := []struct {
		desc string
		src  string
		want string
	}{
		{
			desc: "simple",
			src: `
foo = "123"
`,
			want: "123",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			f, diags := hclwrite.ParseConfig([]byte(tc.src), "", hcl.Pos{Line: 1, Column: 1})
			if len(diags) != 0 {
				for _, diag := range diags {
					t.Logf("- %s", diag.Error())
				}
				t.Fatalf("unexpected diagnostics")
			}

			attr := f.Body().GetAttribute("foo")
			got := getAttributeValueAsUnquotedString(attr)

			if got != tc.want {
				t.Errorf("got = %s, but want = %s", got, tc.want)
			}
		})
	}
}

func TestTokensForListPerLine(t *testing.T) {
	cases := []struct {
		desc string
		list []string
		want string
	}{
		{
			desc: "simple",
			list: []string{"aaa", "bbb"},
			want: `foo = [
  "aaa",
  "bbb",
]
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			f := hclwrite.NewEmptyFile()
			f.Body().SetAttributeRaw("foo", tokensForListPerLine(tc.list))

			got := string(hclwrite.Format(f.BuildTokens(nil).Bytes()))

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", got, tc.want, diff)
			}
		})
	}
}
