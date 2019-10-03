package tfupdate

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pkg/errors"
)

// Updater is an interface which updates a version constraint in HCL.
type Updater interface {
	// Update updates a version constraint.
	// Note that this method will rewrite the AST passed as an argument.
	Update(*hclwrite.File) error
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

// UpdateHCL reads HCL from io.Reader, updates version constraints
// and writes updated contents to io.Writer.
// Note that a filename is used only for an error message.
// If contents changed successfully, it returns true, or otherwise returns false.
// If an error occurs, Nothing is written to the output stream.
func UpdateHCL(r io.Reader, w io.Writer, filename string, o Option) (bool, error) {
	input, err := ioutil.ReadAll(r)
	if err != nil {
		return false, fmt.Errorf("failed to read input: %s", err)
	}

	f, diags := hclwrite.ParseConfig(input, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return false, fmt.Errorf("failed to parse input: %s", diags)
	}

	u, err := NewUpdater(o)
	if err != nil {
		return false, err
	}

	u.Update(f)
	output := f.BuildTokens(nil).Bytes()

	if _, err := w.Write(output); err != nil {
		return false, fmt.Errorf("failed to write output: %s", err)
	}

	isUpdated := !bytes.Equal(input, output)
	return isUpdated, nil
}
