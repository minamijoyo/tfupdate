package tfupdate

import (
	"context"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// OpenTofuUpdater is a updater implementation which updates the OpenTofu version constraint.
type OpenTofuUpdater struct {
	version string
}

// NewOpenTofuUpdater is a factory method which returns an OpenTofuUpdater instance.
func NewOpenTofuUpdater(version string) (Updater, error) {
	if len(version) == 0 {
		return nil, errors.Errorf("failed to new opentofu updater. version is required")
	}

	return &OpenTofuUpdater{
		version: version,
	}, nil
}

// Update updates the OpenTofu version constraint.
// Note that this method will rewrite the AST passed as an argument.
func (u *OpenTofuUpdater) Update(_ context.Context, _ *ModuleContext, filename string, f *hclwrite.File) error {
	if filepath.Base(filename) == ".terraform.lock.hcl" {
		// skip a lock file.
		return nil
	}

	for _, tf := range allMatchingBlocks(f.Body(), "terraform", []string{}) {
		// set a version to attribute value only if the key exists
		if tf.Body().GetAttribute("required_version") != nil {
			tf.Body().SetAttributeValue("required_version", cty.StringVal(u.version))
		}
	}

	return nil
}
