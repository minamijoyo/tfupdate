package tfupdate

import (
	"fmt"
	"regexp"
)

// Option is a set of parameters to update.
type Option struct {
	// A type of updater. Valid values are as follows:
	// - terraform
	// - provider
	// - module
	// - lock
	updateType string

	// If an updateType is terraform, there is no meaning.
	// If an updateType is provider or module, Set a name of provider or module.
	name string

	// a new version constraint
	version string

	// platforms is a list of target platforms to generate hash values.
	// Target platform names consist of an operating system and a CPU
	// architecture such as darwin_arm64.
	platforms []string

	// If a recursive flag is true, it checks and updates directories recursively.
	recursive bool

	// An array of regular expression for paths to ignore.
	ignorePaths []*regexp.Regexp
}

// NewOption returns an option.
func NewOption(updateType string, name string, version string, platforms []string, recursive bool, ignorePaths []string) (Option, error) {
	regexps := make([]*regexp.Regexp, 0, len(ignorePaths))
	for _, ignorePath := range ignorePaths {
		if len(ignorePath) == 0 {
			continue
		}

		r, err := regexp.Compile(ignorePath)
		if err != nil {
			return Option{}, fmt.Errorf("faild to compile regexp for ignorePath: %s", err)
		}
		regexps = append(regexps, r)
	}

	return Option{
		updateType:  updateType,
		name:        name,
		version:     version,
		platforms:   platforms,
		recursive:   recursive,
		ignorePaths: regexps,
	}, nil
}

// MatchIgnorePaths returns whether any of the ignore conditions are met.
func (o *Option) MatchIgnorePaths(path string) bool {
	for _, r := range o.ignorePaths {
		if r.MatchString(path) {
			return true
		}
	}

	return false
}
