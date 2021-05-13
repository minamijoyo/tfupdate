package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/release"
	flag "github.com/spf13/pflag"
)

// ReleaseLatestCommand is a command which gets the latest release version.
type ReleaseLatestCommand struct {
	Meta
	sourceType string
	source     string
}

// Run runs the procedure of this command.
func (c *ReleaseLatestCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("release latest", flag.ContinueOnError)
	cmdFlags.StringVarP(&c.sourceType, "source-type", "s", "github", "A type of release data source")

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 1 {
		c.UI.Error(fmt.Sprintf("The command expects 1 argument, but got %d", len(cmdFlags.Args())))
		c.UI.Error(c.Help())
		return 1
	}

	c.source = cmdFlags.Arg(0)

	r, err := newRelease(c.sourceType, c.source)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	v, err := release.Latest(context.Background(), r)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(v)
	return 0
}

// Help returns long-form help text.
func (c *ReleaseLatestCommand) Help() string {
	helpText := `
Usage: tfupdate release latest [options] <SOURCE>

Arguments
  SOURCE             A path of release data source.
                     Valid format depends on --source-type option.
                       - github or gitlab:
                         owner/repo
                         e.g. terraform-providers/terraform-provider-aws
                      - tfregistryModule
                         namespace/name/provider
                         e.g. terraform-aws-modules/vpc/aws
                      - tfregistryProvider (experimental)
                         namespace/type
                         e.g. hashicorp/aws

Options:
  -s  --source-type  A type of release data source.
                     Valid values are
                       - github (default)
                       - gitlab
                       - tfregistryModule
                       - tfregistryProvider (experimental)
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ReleaseLatestCommand) Synopsis() string {
	return "Get the latest release version"
}
