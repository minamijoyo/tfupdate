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

		// set a version to attribute value only if the key exists
		if p.Body().GetAttribute(u.name) != nil {
			p.Body().SetAttributeValue(u.name, cty.StringVal(u.version))
		}
	}

	return nil
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
