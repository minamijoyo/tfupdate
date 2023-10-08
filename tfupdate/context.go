package tfupdate

import (
	"log"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/spf13/afero"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// GlobalContext is information that is shared over the lifetime of the process.
type GlobalContext struct {
	// fs is an afero filesystem for testing.
	fs afero.Fs

	// updater is an interface to rewriting rule implementations.
	updater Updater

	// option is a set of global parameters.
	option Option
}

// NewGlobalContext returns a new instance of NewGlobalContext.
func NewGlobalContext(fs afero.Fs, o Option) (*GlobalContext, error) {
	updater, err := NewUpdater(o)
	if err != nil {
		return nil, err
	}

	gc := &GlobalContext{
		fs:      fs,
		updater: updater,
		option:  o,
	}

	return gc, nil
}

// ModuleContext is information shared across files within a directory.
type ModuleContext struct {
	// gc is a pointer to delegate some implementations to GlobalContext.
	gc *GlobalContext

	// dir is a relative path to the module from the current working directory.
	dir string

	// requiredProviders is version constraints of Terraform providers.
	// This is the result of parsing terraform-config-inspect and may contain
	// multiple constraints. The meaning depends on the use case and is therefore
	// lazily evaluated.
	requiredProviders map[string]*tfconfig.ProviderRequirement
}

// SelectedProvider is the source address and version of the provider, as
// inferred from the version constraint.
type SelectedProvider struct {
	// source is a source address of the provider.
	Source string

	// version is a version of the provider.
	Version string
}

// aferoToTfconfigFS converts afero.Fs to tfconfig.FS.
// The filesystem has been replaced for testing purposes, but due to historical
// reasons, we use afero.Fs instead of standard io/fs.FS introduced in Go 1.16.
// On the other hand, the tfconfig uses its own tfconfig.FS, which is also
// incompatible the standard one. Fortunately, both have adaptors for
// converting the interface to the standard one. Converting afero.Fs to
// io/fs.FS and then to tfconfig.FS makes the types match.
// Note that the standard io/fs.FS doesn't support any write operations and
// afero.IOFS doesn't support absolute paths at the time of writing.
// It might be better to use the native OS filesystem for testing without
// relying on afero.
func aferoToTfconfigFS(afs afero.Fs) tfconfig.FS {
	return tfconfig.WrapFS(afero.NewIOFS(afs))
}

// NewModuleContext parses a given module and returns a new ModuleContext.
// The dir is a relative path to the module from the current working directory.
func NewModuleContext(dir string, gc *GlobalContext) (*ModuleContext, error) {
	requiredProviders := make(map[string]*tfconfig.ProviderRequirement)
	m, diags := tfconfig.LoadModuleFromFilesystem(aferoToTfconfigFS(gc.fs), dir)
	if diags.HasErrors() {
		// There is a known issue passing absolute paths to afero.IOFS results in
		// an error, but as the result of module inspection is not essential for
		// all use cases now, we intentionally ignore the error here.
		// https://github.com/minamijoyo/tfupdate/issues/93
		log.Printf("[DEBUG] failed to load module: dir = %s, err = %s", dir, diags)
	} else {
		requiredProviders = m.RequiredProviders
	}

	c := &ModuleContext{
		gc:                gc,
		dir:               dir,
		requiredProviders: requiredProviders,
	}

	return c, nil
}

// GlobalContext returns an instance of the global context.
func (mc *ModuleContext) GlobalContext() *GlobalContext {
	return mc.gc
}

// FS returns an instance of afero filesystem
func (mc *ModuleContext) FS() afero.Fs {
	return mc.gc.fs
}

// Updater returns an instance of Updater.
func (mc *ModuleContext) Updater() Updater {
	return mc.gc.updater
}

// Option returns an instance of Option.
func (mc *ModuleContext) Option() Option {
	return mc.gc.option
}

// SelectedProviders returns a list of providers inferred from version constraints.
// The result is sorted alphabetically by source address.
// Version constraints only support simple constants and not comparison
// operators. Ignore what cannot be interpreted.
func (mc *ModuleContext) SelecetedProviders() []SelectedProvider {
	selected := make(map[string]string)
	for _, p := range mc.requiredProviders {
		if p.Source == "" {
			// A source address with an empty string implies an unknown namespace prior to
			// Terraform v0.13, but since this is already a deprecated usage, we don't
			// implicitly complement the official hashicorp namespace and is not included
			// in the results.
			log.Printf("[DEBUG] ModuleContext.SelecetedProviders: ignore legacy provider address notation: %s", p.Source)
			continue
		}

		v := selectVersion(p.VersionConstraints)

		if v == "" {
			// Ignore if no version is specified.
			log.Printf("[DEBUG] ModuleContext.SelecetedProviders: ignore no version selected: %s", p.Source)
			continue
		}

		// It is not possible to mix multiple provider versions in one module, so
		// simply overwrite without taking duplicates into account
		selected[p.Source] = v
	}

	// Sort to get stable results
	keys := maps.Keys(selected)
	slices.Sort(keys)

	ret := []SelectedProvider{}
	for _, k := range keys {
		s := SelectedProvider{Source: k, Version: selected[k]}
		ret = append(ret, s)
	}
	return ret
}

// selectVersion resolves version constraints and returns the version.
// Note that it does not actually re-implement the resolution of version
// constraints in terraform init. It is very simplified for the use we need.
// Version constraints only support simple constants and not comparison
// operators. Ignore what cannot be interpreted.
func selectVersion(constraints []string) string {
	for _, c := range constraints {
		v, err := version.NewVersion(c)
		if err != nil {
			// Ignore parse error
			log.Printf("[DEBUG] selectVersion: ignore version parse error: constaraints = %#v, err = %s", constraints, err)
			continue
		}
		// return the first one found
		return v.String()
	}
	return ""
}

// ResolveProviderShortNameFromSource is a helper function to resolve provider
// short names from the source address.
// If not found, return an empty string.
func (mc *ModuleContext) ResolveProviderShortNameFromSource(source string) string {
	for k, v := range mc.requiredProviders {
		if v.Source == source {
			return k
		}
	}

	return ""
}
