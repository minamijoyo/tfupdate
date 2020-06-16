package tfupdate

import (
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
			value, err := attributeToValue(attr)
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
	if _, ok := m["version"]; ok {
		m["version"] = cty.StringVal(u.version)
		p.Body().SetAttributeValue(u.name, cty.ObjectVal(m))
	}
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
