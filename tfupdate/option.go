package tfupdate

import (
	"fmt"
	"regexp"
)

// Option is a set of parameters to update.
type Option struct {
	// A type of updater. Valid value is terraform or provider.
	updateType string

	// A target to be updated.
	// If an updateType is terraform, Set a version.
	// If an updateType is provider, Set a name@version.
	target string

	// If a recursive flag is true, it checks and updates directories recursively.
	recursive bool

	// An array of regular expression for paths to ignore.
	ignorePaths []*regexp.Regexp
}

// NewOption returns an option.
func NewOption(updateType string, target string, recursive bool, ignorePaths []string) (Option, error) {
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
		target:      target,
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
