package tfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	updater, lockMode := mc.Updater().(*LockUpdater)
	fileError := func(err error) error {
		if lockMode {
			return updater.lockfileError(mc, err)
		}
		return err
	}
	r, err := mc.FS().Open(filename)
	if err != nil {
		return fileError(fmt.Errorf("failed to open file: %w", err))
	}

	w := &bytes.Buffer{}
	isUpdated, err := UpdateHCL(ctx, mc, r, w, filename)
	closeErr := r.Close()
	if err != nil {
		return fileError(err)
	}
	if lockMode && closeErr != nil {
		return fileError(fmt.Errorf("failed to close input: %w", closeErr))
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
		if updater, ok := mc.Updater().(*LockUpdater); ok {
			return updater.persistLockfile(mc, filename, result)
		}
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
		if updater, ok := current.Updater().(*LockUpdater); ok {
			return updater.lockfileError(current, fmt.Errorf("failed to open dir: %w", err))
		}
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
		// nolint:staticcheck // QF1001: could apply De Morgan's law
		// Ignore it in favor of subjective readability.
		if !(filepath.Ext(entry.Name()) == ".tf" || filepath.Ext(entry.Name()) == ".tofu" || entry.Name() == ".terraform.lock.hcl") {
			// skip unsupported file type
			continue
		}

		// Lock planning already inspected module sources. Re-reading them after
		// committing the lockfile could report a late, non-transactional error.
		if _, lockMode := current.Updater().(*LockUpdater); lockMode && entry.Name() != ".terraform.lock.hcl" {
			continue
		}

		err := UpdateFile(ctx, current, path)
		if err != nil {
			return err
		}
	}
	if updater, ok := current.Updater().(*LockUpdater); ok {
		return updater.createMissingLockfile(ctx, current, dirname)
	}

	return nil
}

// UpdateFileOrDir updates version constraints in a given file or directory.
func UpdateFileOrDir(ctx context.Context, gc *GlobalContext, path string) error {
	isDir, err := afero.IsDir(gc.fs, path)
	if err != nil {
		if updater, ok := gc.updater.(*LockUpdater); ok {
			dir := path
			if filepath.Base(path) == ".terraform.lock.hcl" {
				dir = filepath.Dir(path)
			}
			mc, contextErr := NewModuleContext(dir, gc)
			if contextErr != nil {
				return fmt.Errorf("module %s: failed to open path: %w", dir, err)
			}
			return updater.lockfileError(mc, fmt.Errorf("failed to open path: %w", err))
		}
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

// createMissingLockfile writes only after resolution and checksum work succeeds.
func (u *LockUpdater) createMissingLockfile(ctx context.Context, mc *ModuleContext, dirname string) error {
	filename := filepath.Join(dirname, ".terraform.lock.hcl")
	option := mc.Option()
	if option.MatchIgnorePaths(filename) {
		return nil
	}
	exists, err := afero.Exists(mc.FS(), filename)
	if err != nil {
		return u.lockfileError(mc, fmt.Errorf("check lockfile: %w", err))
	}
	if exists {
		return nil
	}
	f := hclwrite.NewEmptyFile()
	if err := u.updateLockfile(ctx, mc, f); err != nil {
		return err
	}
	if len(f.Body().Blocks()) == 0 {
		return nil
	}
	contents := hclwrite.Format(f.BuildTokens(nil).Bytes())
	return u.persistLockfile(mc, filename, contents)
}

// persistLockfile attaches the identities available at the storage boundary.
func (u *LockUpdater) persistLockfile(mc *ModuleContext, filename string, contents []byte) error {
	if err := writeLockfileAtomically(mc.FS(), filename, contents); err != nil {
		return u.lockfileError(mc, fmt.Errorf("write lockfile: %w", err))
	}
	return nil
}

// lockfileError also applies before parsing or planning can add identities.
func (u *LockUpdater) lockfileError(mc *ModuleContext, err error) error {
	groups := u.collectProviderConstraints(mc)
	if len(groups) > 0 {
		addresses := make([]string, len(groups))
		for i, group := range groups {
			addresses[i] = group.address
		}
		return fmt.Errorf("module %s provider %s: %w", mc.dir, strings.Join(addresses, ", "), err)
	}
	return fmt.Errorf("module %s: %w", mc.dir, err)
}

// writeLockfileAtomically keeps the destination intact until all output has
// been written and closed. A sibling temporary file permits atomic renaming.
func writeLockfileAtomically(fs afero.Fs, filename string, contents []byte) (err error) {
	mode := os.FileMode(0644)
	info, err := fs.Stat(filename)
	if err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	temporary, err := afero.TempFile(fs, filepath.Dir(filename), ".tfupdate-lock-")
	if err != nil {
		return err
	}
	closed, committed := false, false
	defer func() {
		if !closed {
			err = errors.Join(err, temporary.Close())
		}
		if !committed {
			err = errors.Join(err, fs.Remove(temporary.Name()))
		}
	}()
	written, err := temporary.Write(contents)
	if err != nil {
		return err
	}
	if written != len(contents) {
		return io.ErrShortWrite
	}
	if err := fs.Chmod(temporary.Name(), mode); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	err = temporary.Close()
	closed = true
	if err != nil {
		return err
	}
	if err := fs.Rename(temporary.Name(), filename); err != nil {
		return err
	}
	committed = true
	return nil
}
