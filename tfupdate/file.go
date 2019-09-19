package tfupdate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/spf13/afero"
)

// UpdateFile updates version constraints in a single file.
// We use an afero filesystem here for testing.
func UpdateFile(fs afero.Fs, filename string, o Option) error {
	r, err := fs.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	defer r.Close()

	w := &bytes.Buffer{}
	isUpdated, err := UpdateHCL(r, w, filename, o)
	if err != nil {
		return err
	}

	// Write contents back to source file if changed.
	if isUpdated {
		updated := w.Bytes()
		// We should be able to choose whether to format output or not.
		// However, the current implementation of (*hclwrite.Body).SetAttributeValue()
		// does not seem to preserve an original SpaceBefore value of attribute.
		// So, we need to format output here.
		result := hclwrite.Format(updated)
		if err = afero.WriteFile(fs, filename, result, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write file: %s", err)
		}
	}

	return nil
}

// UpdateDir updates version constraints for files in a given directory.
// If a recursive flag is true, it checks and updates recursively.
// skip hidden directories such as .terraform or .git.
// It also skips a file without .tf extension.
func UpdateDir(fs afero.Fs, dirname string, recursive bool, o Option) error {
	dir, err := afero.ReadDir(fs, dirname)
	if err != nil {
		return fmt.Errorf("failed to open dir: %s", err)
	}

	for _, entry := range dir {
		path := filepath.Join(dirname, entry.Name())

		if entry.IsDir() {
			// if entry is a directory
			if !recursive {
				// skip directory if a recursive flag is false
				continue
			}
			if strings.HasPrefix(entry.Name(), ".") {
				// skip hidden directories such as .terraform or .git
				continue
			}

			err := UpdateDir(fs, path, recursive, o)
			if err != nil {
				return err
			}

		} else {
			// if entry is a file
			if filepath.Ext(entry.Name()) != ".tf" {
				// skip a file without .tf extension.
				continue
			}

			err := UpdateFile(fs, path, o)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
