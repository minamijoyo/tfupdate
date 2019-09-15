package command

import (
	"github.com/mitchellh/cli"
	"github.com/spf13/afero"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	// UI is a user interface representing input and output.
	UI cli.Ui

	// Fs is an afero filesystem.
	Fs afero.Fs
}
