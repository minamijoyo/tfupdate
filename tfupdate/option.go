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

	// A regular expression for paths to ignore.
	ignorePath *regexp.Regexp
}

// NewOption returns an option.
func NewOption(updateType string, target string, recursive bool, ignorePath string) (Option, error) {
	var ignorePathRegex *regexp.Regexp
	if len(ignorePath) != 0 {
		var err error
		ignorePathRegex, err = regexp.Compile(ignorePath)
		if err != nil {
			return Option{}, fmt.Errorf("faild to compile regexp for ignorePath: %s", err)
		}
	}

	return Option{
		updateType: updateType,
		target:     target,
		recursive:  recursive,
		ignorePath: ignorePathRegex,
	}, nil
}
