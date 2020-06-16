package tfupdate

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
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

// attributeToValue extracts cty.Value from hclwrite.Attribute.
// At the time of writing, there is no way to do with the hclwrite AST,
// so we build low-level byte sequences and parse an expression as a
// hclsyntax.Expression on memory.
func attributeToValue(a *hclwrite.Attribute) (cty.Value, error) {
	// build low-level byte sequences
	src := a.Expr().BuildTokens(nil).Bytes()

	// parse an expression as a hclsyntax.Expression
	expr, diags := hclsyntax.ParseExpression(src, "generated_by_attributeToValue", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return cty.NilVal, fmt.Errorf("failed to parse expression: %s", diags)
	}

	// Get value from expression.
	// We don't need interpolation for any variables and functions here,
	// so we just pass an empty context.
	v, diags := expr.Value(&hcl.EvalContext{})
	if diags.HasErrors() {
		return cty.NilVal, fmt.Errorf("failed to get cty.Value: %s", diags)
	}

	return v, nil
}
