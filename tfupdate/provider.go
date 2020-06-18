package tfupdate

import (
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
		return nil, errors.Errorf("failed to new provider updater. name is required.")
	}

	if len(version) == 0 {
		return nil, errors.Errorf("failed to new provider updater. version is required.")
	}

	return &ProviderUpdater{
		name:    name,
		version: version,
	}, nil
}

// Update updates the provider version constraint.
// Note that this method will rewrite the AST passed as an argument.
func (u *ProviderUpdater) Update(f *hclwrite.File) error {
	if err := u.updateTerraformBlock(f); err != nil {
		return err
	}

	if err := u.updateProviderBlock(f); err != nil {
		return err
	}

	return nil
}

func (u *ProviderUpdater) updateTerraformBlock(f *hclwrite.File) error {
	for _, tf := range allMatchingBlocks(f.Body(), "terraform", []string{}) {
		p := tf.Body().FirstMatchingBlock("required_providers", []string{})
		if p == nil {
			continue
		}

		attr := p.Body().GetAttribute(u.name)
		if attr != nil {
			value, err := getAttributeValue(attr)
			if err != nil {
				return err
			}

			// There are some variations on the syntax of required_providers.
			// So we check a type of value and switch implementations.
			switch {
			case value.Type().IsObjectType():
				u.updateTerraformRequiredProvidersBlockAsObject(p, value)

			case value.Type() == cty.String:
				u.updateTerraformRequiredProvidersBlockAsString(p)

			default:
				return errors.Errorf("failed to update required_providers. unknown type: %#v", value)
			}
		}
	}

	return nil
}

func (u *ProviderUpdater) updateTerraformRequiredProvidersBlockAsObject(p *hclwrite.Block, value cty.Value) {
	// terraform {
	//   required_providers {
	//     aws = {
	//       source  = "hashicorp/aws"
	//       version = "2.65.0"
	//     }
	//   }
	// }
	m := value.AsValueMap()
	if _, ok := m["version"]; !ok {
		// If the version key is missing, just ignore it.
		return
	}

	// Updating the whole object loses original sort order and comments.
	// At the time of writing, there is no way to update a value inside an
	// object directly while preserving original tokens.
	//
	// m["version"] = cty.StringVal(u.version)
	// p.Body().SetAttributeValue(u.name, cty.ObjectVal(m))
	//
	// Since we fully understand the valid syntax, we compromise and read the
	// tokens in order, updating the bytes directly.
	// It's apparently a fragile dirty hack, but I didn't come up with the better
	// way to do this.
	attr := p.Body().GetAttribute(u.name)
	tokens := attr.Expr().BuildTokens(nil)

	i := 0
	// find key of version
	for !(tokens[i].Type == hclsyntax.TokenIdent && string(tokens[i].Bytes) == "version") {
		i++
	}

	// find =
	for tokens[i].Type != hclsyntax.TokenEqual {
		i++
	}

	// find value of old version
	oldVersion := m["version"].AsString()
	for !(tokens[i].Type == hclsyntax.TokenQuotedLit && string(tokens[i].Bytes) == oldVersion) {
		i++
	}

	// Since I've checked for the existence of the version key in advance,
	// if we reach here, we found the token to be updated.
	// So we now update bytes of the token in place.
	tokens[i].Bytes = []byte(u.version)

	return
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
