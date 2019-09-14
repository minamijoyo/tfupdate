package tfupdate

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// UpdateFile updates version constraints in a single file.
func UpdateFile(filename string, o Option) error {
	log.Printf("[DEBUG] Open file: %s", filename)
	r, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	defer r.Close()

	w := &bytes.Buffer{}
	err = UpdateHCL(r, w, filename, o)
	if err != nil {
		return err
	}

	// Write contents back to source file if changed.
	result := w.Bytes()
	if len(result) > 0 {
		log.Printf("[DEBUG] Detect changes. Write file: %s", filename)
		if err = ioutil.WriteFile(filename, result, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write file: %s", err)
		}
	} else {
		log.Printf("[DEBUG] No changes. Skip writing file: %s", filename)
	}

	return nil
}
