package release

// Release is an interface which provides version information.
type Release interface {
	// Latest returns a latest version.
	Latest() (string, error)
}
