package tfupdate

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
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

// getAttributeValueAsString returns a value of Attribute as string.
// There is no way to get value as string directly,
// so we parses tokens of Attribute and build string representation.
// The returned value is unquoted.
func getAttributeValueAsUnquotedString(attr *hclwrite.Attribute) string {
	// find TokenEqual
	expr := attr.Expr()
	exprTokens := expr.BuildTokens(nil)

	// TokenIdent records SpaceBefore, but we should ignore it here.
	quotedValue := strings.TrimSpace(string(exprTokens.Bytes()))

	// unquote
	value := strings.Trim(quotedValue, "\"")

	return value
}

// tokensForListPerLine builds a hclwrite.Tokens for a given list, but breaks the line for each element.
func tokensForListPerLine(list []string) hclwrite.Tokens {
	// The original TokensForValue implementation does not break line by line for list,
	// so we build a token sequence by ourselves.
	tokens := hclwrite.Tokens{}
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenOBrack, Bytes: []byte{'['}})
	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})

	for _, i := range list {
		ts := hclwrite.TokensForValue(cty.StringVal(i))
		tokens = append(tokens, ts...)
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte{','}})
		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte{']'}})

	return tokens
}
