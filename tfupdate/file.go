package tfupdate

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl2/hclwrite"
)

// UpdateFile updates version constraints in a single file.
func UpdateFile(filename string, o Option) error {
	r, err := os.Open(filename)
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
		if err = ioutil.WriteFile(filename, result, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write file: %s", err)
		}
	}

	return nil
}
