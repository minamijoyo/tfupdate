package tfupdate

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// ModuleUpdater is a updater implementation which updates the module version constraint.
type ModuleUpdater struct {
	name    string
	version string
}

// NewModuleUpdater is a factory method which returns an ModuleUpdater instance.
func NewModuleUpdater(name string, version string) (Updater, error) {
	if len(name) == 0 {
		return nil, errors.Errorf("failed to new module updater. name is required.")
	}

	if len(version) == 0 {
		return nil, errors.Errorf("failed to new module updater. version is required.")
	}

	return &ModuleUpdater{
		name:    name,
		version: version,
	}, nil
}

// Update updates the module version constraint.
// Note that this method will rewrite the AST passed as an argument.
func (u *ModuleUpdater) Update(f *hclwrite.File) error {
	if err := u.updateModuleBlock(f); err != nil {
		return err
	}

	return nil
}

func (u *ModuleUpdater) updateModuleBlock(f *hclwrite.File) error {
	for _, m := range allMatchingBlocksByType(f.Body(), "module") {
		if s := m.Body().GetAttribute("source"); s != nil {
			source := parseModuleSorce(s)

			if source == u.name {
				// set a version to attribute value only if the key exists
				if m.Body().GetAttribute("version") != nil {
					m.Body().SetAttributeValue("version", cty.StringVal(u.version))
				}
			}
		}
	}

	return nil
}

func parseModuleSorce(a *hclwrite.Attribute) string {
	tokens := a.Expr().BuildTokens(nil)
	if len(tokens) == 3 &&
		tokens[0].Type == hclsyntax.TokenOQuote &&
		tokens[1].Type == hclsyntax.TokenQuotedLit &&
		tokens[2].Type == hclsyntax.TokenCQuote {
		source := string(tokens[1].Bytes)
		return source
	}
	return ""
}
