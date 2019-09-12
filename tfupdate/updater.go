package tfupdate

import (
	"strings"

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
	// A type of updater. Valid value is terraform or provider.
	updateType string
	// A target to be updated.
	// If an updateType is terraform, Set a version.
	// If an updateType is provider, Set a name@version.
	target string
}

// NewUpdater is a factory method which returns an Updater implementation.
func NewUpdater(o Option) (Updater, error) {
	switch o.updateType {
	case "terraform":
		return NewTerraformUpdater(o.target)
	case "provider":
		s := strings.Split(o.target, "@")
		return &ProviderUpdater{
			name:    s[0],
			version: s[1],
		}, nil
	case "module":
		return nil, errors.Errorf("failed to new updater. module is not currently supported.")
	default:
		return nil, errors.Errorf("failed to new updater. unknown type: %s", o.updateType)
	}
}

// NewOption returns an option.
func NewOption(updateType string, target string) Option {
	return Option{
		updateType: updateType,
		target:     target,
	}
}
