package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func main() {
	filename := "./main.tf"
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file: %s, err = %+v\n", filename, err)
		os.Exit(1)
	}

	f, diags := hclwrite.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		fmt.Fprintf(os.Stderr, "failed to parse file: %s", diags)
		os.Exit(1)
	}

	tf := f.Body().FirstMatchingBlock("terraform", []string{})
	tf.Body().SetAttributeValue("required_version", cty.StringVal("0.12.6"))

	providers := tf.Body().FirstMatchingBlock("required_providers", []string{})
	providers.Body().SetAttributeValue("null", cty.StringVal("2.1.2"))

	aws := f.Body().FirstMatchingBlock("provider", []string{"aws"})
	aws.Body().SetAttributeValue("version", cty.StringVal("2.23.0"))

	tokens := f.BuildTokens(nil)
	buf := hclwrite.Format(tokens.Bytes())

	fmt.Printf("%s\n", buf)
}
