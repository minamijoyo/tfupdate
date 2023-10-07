package tfupdate

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// ProviderUpdater is a updater implementation which updates the provider version constraint.
type ProviderUpdater struct {
	name    string
	version string
}

// NewProviderUpdater is a factory method which returns an ProviderUpdater instance.
func NewProviderUpdater(name string, version string) (Updater, error) {
	if len(name) == 0 {
		return nil, errors.Errorf("failed to new provider updater. name is required")
	}

	if len(version) == 0 {
		return nil, errors.Errorf("failed to new provider updater. version is required")
	}

	return &ProviderUpdater{
		name:    name,
		version: version,
	}, nil
}

// Update updates the provider version constraint.
// Note that this method will rewrite the AST passed as an argument.
func (u *ProviderUpdater) Update(_ context.Context, mc *ModuleContext, filename string, f *hclwrite.File) error {
	if filepath.Base(filename) == ".terraform.lock.hcl" {
		// skip a lock file.
		return nil
	}

	if err := u.updateTerraformBlock(mc, f); err != nil {
		return err
	}

	return u.updateProviderBlock(f)
}

func (u *ProviderUpdater) updateTerraformBlock(mc *ModuleContext, f *hclwrite.File) error {
	for _, tf := range allMatchingBlocks(f.Body(), "terraform", []string{}) {
		p := tf.Body().FirstMatchingBlock("required_providers", []string{})
		if p == nil {
			continue
		}

		name := u.name
		// If the name contains /, assume that a namespace is intended and check the source.
		if strings.Contains(u.name, "/") {
			name = mc.ResolveProviderShortNameFromSource(u.name)
			if name == "" {
				continue
			}
		}

		// The hclwrite.Attribute doesn't have enough AST for object type to check.
		// Get the attribute as a native hcl.Attribute as a compromise.
		hclAttr, err := getHCLNativeAttribute(p.Body(), name)
		if err != nil {
			return err
		}

		if hclAttr != nil {
			// There are some variations on the syntax of required_providers.
			// So we check a type of the value and switch implementations.
			// If the expression can be parsed as a static expression and it's type is a primitive,
			// then it's a legacy string syntax.
			if expr, err := hclAttr.Expr.Value(nil); err == nil && expr.Type().IsPrimitiveType() {
				u.updateTerraformRequiredProvidersBlockAsString(p)
			} else {
				// Otherwise, it's an object syntax.
				if err := u.updateTerraformRequiredProvidersBlockAsObject(p, name, hclAttr); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (u *ProviderUpdater) updateTerraformRequiredProvidersBlockAsObject(p *hclwrite.Block, name string, hclAttr *hcl.Attribute) error {
	// terraform {
	//   required_providers {
	//     aws = {
	//       source  = "hashicorp/aws"
	//       version = "2.65.0"
	//
	//       configuration_aliases = [
	//         aws.primary,
	//         aws.secondary,
	//       ]
	//     }
	//   }
	// }

	oldVersion, err := detectVersionInObject(hclAttr)
	if err != nil {
		return err
	}

	if len(oldVersion) == 0 {
		// If the version key is missing, just ignore it.
		return nil
	}

	// Updating the whole object loses original sort order and comments.
	// At the time of writing, there is no way to update a value inside an
	// object directly while preserving original tokens.
	//
	// Since we fully understand the valid syntax, we compromise and read the
	// tokens in order, updating the bytes directly.
	// It's apparently a fragile dirty hack, but I didn't come up with the better
	// way to do this.
	attr := p.Body().GetAttribute(name)
	tokens := attr.Expr().BuildTokens(nil)

	i := 0
	// find key of version
	// Although not explicitly stated in the required_providers documentation,
	// a TokenQuotedLit is also valid token. Strict speaking there are more
	// variants because the left hand side of object key accepts an expression in
	// HCL. For accurate implementation, it should be implemented using the
	// original parser.
	for !((tokens[i].Type == hclsyntax.TokenIdent || tokens[i].Type == hclsyntax.TokenQuotedLit) &&
		string(tokens[i].Bytes) == "version") {
		i++
	}

	// find =
	for tokens[i].Type != hclsyntax.TokenEqual {
		i++
	}

	// find value of old version
	for !(tokens[i].Type == hclsyntax.TokenQuotedLit && string(tokens[i].Bytes) == oldVersion) {
		i++
	}

	// Since I've checked for the existence of the version key in advance,
	// if we reach here, we found the token to be updated.
	// So we now update bytes of the token in place.
	tokens[i].Bytes = []byte(u.version)

	return nil
}

// detectVersionInObject parses an object expression and detects a value for
// the "version" key.
// If the version key is missing, just returns an empty string without an error.
func detectVersionInObject(hclAttr *hcl.Attribute) (string, error) {
	// The configuration_aliases syntax isn't directly related version updateing,
	// but it contains provider references and causes an parse error without an EvalContext.
	// So we treat the expression as a hcl.ExprMap to avoid fully decoding the object.
	kvs, diags := hcl.ExprMap(hclAttr.Expr)
	if diags.HasErrors() {
		return "", fmt.Errorf("failed to parse expr as hcl.ExprMap: %s", diags)
	}

	oldVersion := ""
	for _, kv := range kvs {
		key, diags := kv.Key.Value(nil)
		if diags.HasErrors() {
			return "", fmt.Errorf("failed to get key: %s", diags)
		}
		if key.AsString() == "version" {
			value, diags := kv.Value.Value(nil)
			if diags.HasErrors() {
				return "", fmt.Errorf("failed to get value: %s", diags)
			}
			oldVersion = value.AsString()
		}
	}

	return oldVersion, nil
}

func (u *ProviderUpdater) updateTerraformRequiredProvidersBlockAsString(p *hclwrite.Block) {
	// terraform {
	//   required_providers {
	//     aws = "2.65.0"
	//   }
	// }
	p.Body().SetAttributeValue(u.name, cty.StringVal(u.version))
}

func (u *ProviderUpdater) updateProviderBlock(f *hclwrite.File) error {
	for _, p := range allMatchingBlocks(f.Body(), "provider", []string{u.name}) {
		// set a version to attribute value only if the key exists
		if p.Body().GetAttribute("version") != nil {
			p.Body().SetAttributeValue("version", cty.StringVal(u.version))
		}
	}

	return nil
}
