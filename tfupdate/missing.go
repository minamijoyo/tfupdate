package tfupdate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"
)

type versionFile struct {
	path     string
	original []byte
	config   *hclwrite.File
}

type versionWrite struct {
	path     string
	contents []byte
}

// versionPlan separates inspection from writes across the entire invocation.
type versionPlan struct {
	fs     afero.Fs
	option Option
	writes []versionWrite
}

func updateMissingVersions(gc *GlobalContext, path string) error {
	p := &versionPlan{fs: gc.fs, option: gc.option}
	path = filepath.Clean(path)
	if p.option.MatchIgnorePaths(path) {
		return nil
	}
	info, err := p.fs.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect path %s: %w", path, err)
	}
	if err == nil && info.IsDir() {
		if err := p.directory(path); err != nil {
			return err
		}
	} else {
		missing := os.IsNotExist(err)
		if missing && !nativeVersionFile(path) {
			return fmt.Errorf("cannot create non-native configuration: %s", path)
		}
		if filepath.Base(path) == ".terraform.lock.hcl" {
			return nil
		}
		if err := p.singleFile(path, missing); err != nil {
			return err
		}
	}
	for _, write := range p.writes {
		if err := afero.WriteFile(p.fs, write.path, write.contents, os.ModePerm); err != nil {
			return fmt.Errorf("failed to write file %s: %w", write.path, err)
		}
	}
	return nil
}

func nativeVersionFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".tf" || ext == ".tofu"
}

func (p *versionPlan) read(path string) (*versionFile, error) {
	src, err := afero.ReadFile(p.fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	config, err := safeParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if err != nil {
		return nil, err
	}
	return &versionFile{path: path, original: src, config: config}, nil
}

func (p *versionPlan) entries(dir string) ([]os.FileInfo, error) {
	entries, err := afero.ReadDir(p.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	return entries, nil
}

func (p *versionPlan) inspect(dir string, entries []os.FileInfo) ([]*versionFile, error) {
	var files []*versionFile
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() || !nativeVersionFile(path) || p.option.MatchIgnorePaths(path) {
			continue
		}
		file, err := p.read(path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (p *versionPlan) directory(dir string) error {
	entries, err := p.entries(dir)
	if err != nil {
		return err
	}
	files, err := p.inspect(dir, entries)
	if err != nil {
		return err
	}
	if err := p.module(dir, files); err != nil {
		return err
	}
	if !p.option.recursive {
		return nil
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || p.option.MatchIgnorePaths(path) {
			continue
		}
		if err := p.directory(path); err != nil {
			return err
		}
	}
	return nil
}

func hasVersion(file *versionFile) bool {
	for _, block := range allMatchingBlocks(file.config.Body(), "terraform", nil) {
		if block.Body().GetAttribute("required_version") != nil {
			return true
		}
	}
	return false
}

func moduleHasVersion(files []*versionFile) bool {
	for _, file := range files {
		if hasVersion(file) {
			return true
		}
	}
	return false
}

func (p *versionPlan) module(dir string, files []*versionFile) error {
	if moduleHasVersion(files) {
		for _, file := range files {
			p.update(file)
		}
		return nil
	}
	destination := filepath.Join(dir, "versions.tf")
	if p.option.MatchIgnorePaths(destination) {
		return nil
	}
	for _, file := range files {
		if file.path == destination {
			p.insert(file)
			return nil
		}
	}
	// An existing directory or unreadable destination must never be replaced.
	if _, err := p.fs.Stat(destination); !os.IsNotExist(err) {
		if err != nil {
			return fmt.Errorf("failed to inspect destination %s: %w", destination, err)
		}
		return fmt.Errorf("destination is not a configuration file: %s", destination)
	}
	p.insert(&versionFile{path: destination, config: hclwrite.NewEmptyFile()})
	return nil
}

func (p *versionPlan) singleFile(path string, missing bool) error {
	dir := filepath.Dir(path)
	entries, err := p.entries(dir)
	if err != nil {
		return err
	}
	files, err := p.inspect(dir, entries)
	if err != nil {
		return err
	}
	var target *versionFile
	for _, file := range files {
		if file.path == path {
			target = file
			break
		}
	}
	if target == nil {
		if missing {
			target = &versionFile{path: path, config: hclwrite.NewEmptyFile()}
		} else {
			target, err = p.read(path)
			if err != nil {
				return err
			}
		}
	}
	if hasVersion(target) {
		p.update(target)
	} else if !moduleHasVersion(files) {
		p.insert(target)
	}
	return nil
}

func (p *versionPlan) update(file *versionFile) {
	if !hasVersion(file) {
		return
	}
	for _, block := range allMatchingBlocks(file.config.Body(), "terraform", nil) {
		if block.Body().GetAttribute("required_version") != nil {
			block.Body().SetAttributeValue("required_version", cty.StringVal(p.option.version))
		}
	}
	p.stage(file)
}

func (p *versionPlan) insert(file *versionFile) {
	blocks := allMatchingBlocks(file.config.Body(), "terraform", nil)
	var block *hclwrite.Block
	if len(blocks) > 0 {
		block = blocks[0]
		// A single-line block must become multiline before adding an attribute.
		body := block.Body()
		tokens := body.BuildTokens(nil)
		body.Clear()
		body.AppendNewline()
		body.AppendUnstructuredTokens(tokens)
		body.AppendNewline()
	} else {
		file.config.Body().AppendNewline()
		block = file.config.Body().AppendNewBlock("terraform", nil)
	}
	block.Body().SetAttributeValue("required_version", cty.StringVal(p.option.version))
	p.stage(file)
}

func (p *versionPlan) stage(file *versionFile) {
	rendered := file.config.BuildTokens(nil).Bytes()
	if bytes.Equal(file.original, rendered) {
		return
	}
	formatted := hclwrite.Format(rendered)
	if bytes.Equal(file.original, formatted) {
		return
	}
	p.writes = append(p.writes, versionWrite{path: file.path, contents: formatted})
}
