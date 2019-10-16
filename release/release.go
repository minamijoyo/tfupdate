package release

import (
	"github.com/pkg/errors"
)

// Release is an interface which provides version information.
type Release interface {
	// Latest returns a latest version.
	Latest() (string, error)
}

// NewRelease is a factory method which returns a Release implementation.
func NewRelease(releaseType string, url string) (Release, error) {
	switch releaseType {
	case "github":
		return NewGitHubRelease(url)
	default:
		return nil, errors.Errorf("failed to new release. unknown type: %s", releaseType)
	}
}

// ResolveVersionAlias resolves a version alias.
func ResolveVersionAlias(r Release, alias string) (string, error) {
	switch alias {
	case "latest":
		return r.Latest()
	default:
		// if an alias does not match keywords, just return alias as a version.
		return alias, nil
	}
}
