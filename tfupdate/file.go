package tfupdate

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/afero"
)

// UpdateFile updates version constraints in a single file.
// We use an afero filesystem here for testing.
func UpdateFile(ctx context.Context, mc *ModuleContext, filename string) error {
	log.Printf("[DEBUG] check file: %s", filename)
	r, err := mc.FS().Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	defer r.Close()

	w := &bytes.Buffer{}
	isUpdated, err := UpdateHCL(ctx, mc, r, w, filename)
	if err != nil {
		return err
	}

	// Write contents back to source file if changed.
	if isUpdated {
		log.Printf("[INFO] update file: %s", filename)
		updated := w.Bytes()
		// We should be able to choose whether to format output or not.
		// However, the current implementation of (*hclwrite.Body).SetAttributeValue()
		// does not seem to preserve an original SpaceBefore value of attribute.
		// So, we need to format output here.
		result := hclwrite.Format(updated)
		if err = afero.WriteFile(mc.FS(), filename, result, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write file: %s", err)
		}
	}

	return nil
}

// UpdateDir updates version constraints for files in a given directory.
// If a recursive flag is true, it checks and updates recursively.
// skip hidden directories such as .terraform or .git.
// It also skips unsupported file type.
func UpdateDir(ctx context.Context, current *ModuleContext, dirname string) error {
	log.Printf("[DEBUG] check dir: %s", dirname)
	option := current.Option()
	dir, err := afero.ReadDir(current.FS(), dirname)
	if err != nil {
		return fmt.Errorf("failed to open dir: %s", err)
	}

	for _, entry := range dir {
		path := filepath.Join(dirname, entry.Name())

		// if a path of entry matches ignorePaths, skip it.
		if option.MatchIgnorePaths(path) {
			log.Printf("[DEBUG] ignore: %s", path)
			continue
		}

		if entry.IsDir() {
			// if an entry is a directory
			if !option.recursive {
				// skip directory if a recursive flag is false
				continue
			}
			if strings.HasPrefix(entry.Name(), ".") {
				// skip hidden directories such as .terraform or .git
				continue
			}

			child, err := NewModuleContext(path, current.GlobalContext())
			if err != nil {
				return err
			}

			err = UpdateDir(ctx, child, path)
			if err != nil {
				return err
			}

			continue
		}

		// if an entry is a file
		if !(filepath.Ext(entry.Name()) == ".tf" || entry.Name() == ".terraform.lock.hcl") {
			// skip unsupported file type
			continue
		}

		err := UpdateFile(ctx, current, path)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateFileOrDir updates version constraints in a given file or directory.
func UpdateFileOrDir(ctx context.Context, gc *GlobalContext, path string) error {
	isDir, err := afero.IsDir(gc.fs, path)
	if err != nil {
		return fmt.Errorf("failed to open path: %s", err)
	}

	if isDir {
		// if an entry is a directory
		mc, err := NewModuleContext(path, gc)
		if err != nil {
			return err
		}
		return UpdateDir(ctx, mc, path)
	}

	// if an entry is a file
	// Note that even if only the filename is specified, the directory containing
	// it is read for module context analysis.
	dir := filepath.Dir(path)
	mc, err := NewModuleContext(dir, gc)
	if err != nil {
		return err
	}
	// When the filename is intentionally specified,
	// we should not ignore it by its extension as much as possible.
	return UpdateFile(ctx, mc, path)
}
