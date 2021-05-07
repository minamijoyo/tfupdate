package release

import (
	"context"
)

// Release is an interface which provides version information of a module or provider.
type Release interface {
	// Latest returns the latest version of a module or provider.
	Latest(ctx context.Context) (string, error)
	// List returns a list of versions of a module or provider in semver order.
	// If preRelease is set to false, the result doesn't contain pre-releases.
	List(ctx context.Context, maxLength int, preRelease bool) ([]string, error)
}
