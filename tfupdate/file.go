package tfupdate

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclwrite"
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
	err = update(r, w, filename, o)
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

func update(r io.Reader, w io.Writer, filename string, o Option) error {
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
