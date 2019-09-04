package tfupdate

import (
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// TerraformUpdater is a updater implementation which updates the terraform version constraint.
type TerraformUpdater struct {
	version string
}

// NewTerraformUpdater is a factory method which returns an TerraformUpdater instance.
func NewTerraformUpdater(version string) (Updater, error) {
	if len(version) == 0 {
		return nil, errors.Errorf("failed to new terraform updater. version is required.")
	}

	return &TerraformUpdater{
		version: version,
	}, nil
}

// Update updates the terraform version constraint.
// Note that this method will rewrite the AST passed as an argument.
func (u *TerraformUpdater) Update(f *hclwrite.File) error {
	tf := f.Body().FirstMatchingBlock("terraform", []string{})
	if tf == nil {
		return nil
	}

	// set a version to attribute value only if the key exists
	if tf.Body().GetAttribute("required_version") != nil {
		tf.Body().SetAttributeValue("required_version", cty.StringVal(u.version))
	}

	return nil
}
