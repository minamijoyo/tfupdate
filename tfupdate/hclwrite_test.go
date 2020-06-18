package tfupdate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
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
		t.Run(fmt.Sprintf("%s", test.typeName), func(t *testing.T) {
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

func TestGetAttributeValue(t *testing.T) {
	tests := []struct {
		valueAsString string
		want          cty.Value
		ok            bool
	}{
		{
			want: cty.StringVal("FOO"),
			ok:   true,
		},
		{
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("FOO"),
				"bar": cty.StringVal("BAR"),
			}),
			ok: true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s", test.valueAsString), func(t *testing.T) {
			// build hclwrite.Attribute
			f := hclwrite.NewEmptyFile()
			f.Body().SetAttributeValue("test", test.want)
			attr := f.Body().GetAttribute("test")

			got, err := getAttributeValue(attr)
			if test.ok && err != nil {
				t.Errorf("getAttributeValue() with attr = %s returns unexpected err: %+v", test.want, err)
			}

			if !test.ok && err == nil {
				t.Errorf("getAttributeValue() with attr = %s expects to return an error, but no error", test.want)
			}

			if !got.RawEquals(test.want) {
				t.Errorf("getAttributeValue() with attr = %s returns %#v, but want = %#v", test.want, got, test.want)
			}
		})
	}
}
