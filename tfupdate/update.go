package tfupdate

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
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

// UpdateHCL reads HCL from io.Reader, updates version constraints
// and writes updated contents to io.Writer.
// Note that a filename is used only for an error message.
func UpdateHCL(r io.Reader, w io.Writer, filename string, o Option) error {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read input: %s", err)
	}

	log.Printf("[DEBUG] Parse HCL: %s", filename)
	f, diags := hclwrite.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fmt.Errorf("failed to parse input: %s", diags)
	}

	log.Printf("[DEBUG] Initialize updater: %s, %#v", filename, o)
	u, err := NewUpdater(o)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Update HCL: %s", filename)
	u.Update(f)
	updated := f.BuildTokens(nil).Bytes()

	// Write contents to buffer if changed.
	if !bytes.Equal(src, updated) {
		log.Printf("[DEBUG] Detect changes: %s", filename)
		log.Printf("[DEBUG] Execute fmt: %s", filename)
		result := hclwrite.Format(updated)

		log.Printf("[DEBUG] Write buffer: %s", filename)
		if _, err := w.Write(result); err != nil {
			return fmt.Errorf("failed to write output: %s", err)
		}
	} else {
		log.Printf("[DEBUG] No changes. Skip writing buffer: %s", filename)
	}

	return nil
}
