package command

import (
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/release"
	flag "github.com/spf13/pflag"
)

// ReleaseCommand is a command which gets the latest release version.
type ReleaseCommand struct {
	Meta
	url string
}

// Run runs the procedure of this command.
func (c *ReleaseCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("release", flag.ContinueOnError)

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 1 {
		c.UI.Error(fmt.Sprintf("The command expects 1 argument, but got %#v", cmdFlags.Args()))
		c.UI.Error(c.Help())
		return 1
	}

	c.url = cmdFlags.Arg(0)

	r, err := release.NewRelease("github", c.url)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	v, err := release.ResolveVersionAlias(r, "latest")
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(v)
	return 0
}

// Help returns long-form help text.
func (c *ReleaseCommand) Help() string {
	helpText := `
Usage: tfupdate release [options] <URL>

Arguments
  URL                A URL of the release repository
                     (e.g. https://github.com/terraform-providers/terraform-provider-aws)
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ReleaseCommand) Synopsis() string {
	return "Get the latest release version"
}
