package tfupdate

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// allMatchingBlocks returns all matching blocks from the body that have the
// given name and labels or returns an empty list if there is currently no
// matching block.
func allMatchingBlocks(b *hclwrite.Body, typeName string, labels []string) []*hclwrite.Block {
	var matched []*hclwrite.Block
	for _, block := range b.Blocks() {
		if typeName == block.Type() {
			labelNames := block.Labels()
			if len(labels) == 0 && len(labelNames) == 0 {
				matched = append(matched, block)
				continue
			}
			if reflect.DeepEqual(labels, labelNames) {
				matched = append(matched, block)
			}
		}
	}

	return matched
}

// allMatchingBlocksByType returns all matching blocks from the body that have the
// given name or returns an empty list if there is currently no matching block.
// This method is useful when you want to ignore label differences.
func allMatchingBlocksByType(b *hclwrite.Body, typeName string) []*hclwrite.Block {
	var matched []*hclwrite.Block
	for _, block := range b.Blocks() {
		if typeName == block.Type() {
			matched = append(matched, block)
		}
	}

	return matched
}

// getHCLNativeAttribute gets hclwrite.Attribute as a native hcl.Attribute.
// At the time of writing, there is no way to do with the hclwrite AST,
// so we build low-level byte sequences and parse an attribute as a
// hcl.Attribute on memory.
// If not found, returns nil without an error.
func getHCLNativeAttribute(body *hclwrite.Body, name string) (*hcl.Attribute, error) {
	attr := body.GetAttribute(name)
	if attr == nil {
		return nil, nil
	}

	// build low-level byte sequences
	attrAsBytes := attr.Expr().BuildTokens(nil).Bytes()
	src := append([]byte(name+" = "), attrAsBytes...)

	// parse an expression as a hcl.File.
	// Note that an attribute may contains references, which are defined outside the file.
	// So we cannot simply use hclsyntax.ParseExpression or hclsyntax.ParseConfig here.
	// We need to use a loe-level parser not to resolve all references.
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(src, "generated_by_getHCLNativeAttribute")
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse expression: %s", diags)
	}

	attrs, diags := file.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to get attributes: %s", diags)
	}

	hclAttr, ok := attrs[name]
	if !ok {
		return nil, fmt.Errorf("attribute not found: %s", src)
	}

	return hclAttr, nil
}
