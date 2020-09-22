package release

import (
	"context"
)

// Release is an interface which provides version information of a module or provider.
type Release interface {
	// Latest returns the latest version of a module or provider.
	Latest(ctx context.Context) (string, error)
	// List returns a list of versions of a module or provider.
	List(ctx context.Context, maxLength int) ([]string, error)
}
