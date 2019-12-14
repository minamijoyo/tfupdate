package release

import (
	"context"
)

// Release is an interface which provides version information.
type Release interface {
	// Latest returns a latest version.
	Latest(ctx context.Context) (string, error)
}
