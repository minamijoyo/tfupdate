package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

// ReleaseCommand is a command which just shows help for subcommands.
type ReleaseCommand struct {
	Meta
}

// Run runs the procedure of this command.
func (c *ReleaseCommand) Run(args []string) int { // nolint revive unused-parameter
	return cli.RunResultHelp
}

// Help returns long-form help text.
func (c *ReleaseCommand) Help() string {
	helpText := `
Usage: tfupdate release <subcommand> [options] [args]

  This command has subcommands for release version information.
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ReleaseCommand) Synopsis() string {
	return "Get release version information"
}
