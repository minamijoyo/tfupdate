package release

import "github.com/pkg/errors"

// Release is an interface which provides version information.
type Release interface {
	// Latest returns a latest version.
	Latest() (string, error)
}

// NewRelease is a factory method which returns an Release implementation.
func NewRelease(sourceType string, source string) (Release, error) {
	switch sourceType {
	case "github":
		return NewGitHubRelease(source)
	default:
		return nil, errors.Errorf("failed to new release data source. unknown type: %s", sourceType)
	}
}
