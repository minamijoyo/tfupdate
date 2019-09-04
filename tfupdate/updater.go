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

// Option is a set of parameters to update.
type Option struct {
	updaterType string
	name        string
	version     string
}

// NewUpdater is a factory method which returns an Updater implementation.
func NewUpdater(o Option) (Updater, error) {
	switch o.updaterType {
	case "terraform":
		return NewTerraformUpdater(o.version)
	case "provider":
		return &ProviderUpdater{
			name:    o.name,
			version: o.version,
		}, nil
	case "module":
		return nil, errors.Errorf("failed to new updater. module is not currently supported.")
	default:
		return nil, errors.Errorf("failed to new updater. unknown type: %s", o.updaterType)
	}
}

// NewOption returns an option.
func NewOption(updateType string, name string, version string) Option {
	return Option{
		updaterType: updateType,
		name:        name,
		version:     version,
	}
}
