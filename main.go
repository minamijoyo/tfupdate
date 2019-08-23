package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func main() {
	filename := "./main.tf"

	err := updateFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func updateFile(filename string) error {
	r, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %+v", err)
	}

	err = update(r, filename, os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func update(r io.Reader, filename string, w io.Writer) error {
	f, err := parseHCL(r, filename)
	if err != nil {
		return err
	}

	err = updateHCL(f)
	if err != nil {
		return err
	}

	err = writeHCL(f, w)
	if err != nil {
		return err
	}

	return nil
}

func parseHCL(r io.Reader, filename string) (*hclwrite.File, error) {
	src, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, fmt.Errorf("failed to read input: err = %+v", err)
	}

	f, diags := hclwrite.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse file: %s", diags)
	}

	return f, nil
}

func writeHCL(f *hclwrite.File, w io.Writer) error {
	tokens := f.BuildTokens(nil)
	buf := hclwrite.Format(tokens.Bytes())

	fmt.Fprintln(w, string(buf))

	return nil
}

func updateHCL(f *hclwrite.File) error {
	err := updateTerraform(f)
	if err != nil {
		return err
	}

	err = updateProvider(f)
	if err != nil {
		return err
	}

	return nil
}

func updateTerraform(f *hclwrite.File) error {
	tf := f.Body().FirstMatchingBlock("terraform", []string{})
	tf.Body().SetAttributeValue("required_version", cty.StringVal("0.12.6"))

	return nil
}

func updateProvider(f *hclwrite.File) error {
	tf := f.Body().FirstMatchingBlock("terraform", []string{})
	providers := tf.Body().FirstMatchingBlock("required_providers", []string{})
	providers.Body().SetAttributeValue("null", cty.StringVal("2.1.2"))

	aws := f.Body().FirstMatchingBlock("provider", []string{"aws"})
	aws.Body().SetAttributeValue("version", cty.StringVal("2.23.0"))

	return nil
}
