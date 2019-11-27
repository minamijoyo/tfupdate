package tfupdate

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"runtime/debug"

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
		return NewTerraformUpdater(o.version)
	case "provider":
		return NewProviderUpdater(o.name, o.version)
	case "module":
		return NewModuleUpdater(o.name, o.version)
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

	f, err := safeParseConfig(input, filename, hcl.Pos{Line: 1, Column: 1})
	if err != nil {
		return false, err
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

// safeParseConfig parses config and recovers if panic occurs.
// The current hclwrite implementation is no perfect and will panic if
// unparseable input is given. We just treat it as a parse error so as not to
// surprise users of tfupdate.
func safeParseConfig(src []byte, filename string, start hcl.Pos) (f *hclwrite.File, e error) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[DEBUG] failed to parse input: %s\nstacktrace: %s", filename, string(debug.Stack()))
			// Set a return value from panic recover
			e = fmt.Errorf(`failed to parse input: %s
panic: %s
This may be caused by a bug in the hclwrite parser.
As a workaround, you can ignore this file with --ignore-path option`, filename, err)
		}
	}()

	f, diags := hclwrite.ParseConfig(src, filename, start)

	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse input: %s", diags)
	}

	return f, nil
}
