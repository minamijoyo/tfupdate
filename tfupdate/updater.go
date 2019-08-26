package tfupdate

import (
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/pkg/errors"
)

// Updater is an interface which updates a version constraint in HCL.
type Updater interface {
	// Update updates a version constraint.
	// Note that this method will rewrite the AST passed as an argument.
	Update(*hclwrite.File) error
}

// NewUpdater is a factory method which returns an Updater implementation.
func NewUpdater(updaterType, name string, version string) (Updater, error) {
	switch updaterType {
	case "terraform":
		return NewTerraformUpdater(version)
	case "provider":
		return &ProviderUpdater{
			name:    name,
			version: version,
		}, nil
	case "module":
		return nil, errors.Errorf("failed to new updater. module is not currently supported.")
	default:
		return nil, errors.Errorf("failed to new updater. unknown type: %s", updaterType)
	}
}
