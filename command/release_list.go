package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/release"
	flag "github.com/spf13/pflag"
)

// ReleaseListCommand is a command which gets a list of release versions.
type ReleaseListCommand struct {
	Meta
	maxLength  int
	preRelease bool
	sourceType string
	source     string
}

// Run runs the procedure of this command.
func (c *ReleaseListCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("release list", flag.ContinueOnError)
	cmdFlags.IntVarP(&c.maxLength, "max-length", "n", 10, "the maximum length of list")
	cmdFlags.BoolVar(&c.preRelease, "pre-release", false, "show pre-releases")
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

	versions, err := release.List(context.Background(), r, c.maxLength, c.preRelease)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(strings.Join(versions, "\n"))
	return 0
}

// Help returns long-form help text.
func (c *ReleaseListCommand) Help() string {
	helpText := `
Usage: tfupdate release list [options] <SOURCE>

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

  -n  --max-length   The maximum length of list.
      --pre-release  Show pre-releases. (default: false)
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ReleaseListCommand) Synopsis() string {
	return "Get a list of release versions"
}
