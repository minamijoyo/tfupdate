package command

import (
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/release"
	flag "github.com/spf13/pflag"
)

// ReleaseLatestCommand is a command which gets the latest release version.
type ReleaseLatestCommand struct {
	Meta
	repositoryPath string
}

// Run runs the procedure of this command.
func (c *ReleaseLatestCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("release latest", flag.ContinueOnError)

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 1 {
		c.UI.Error(fmt.Sprintf("The command expects 1 argument, but got %#v", cmdFlags.Args()))
		c.UI.Error(c.Help())
		return 1
	}

	c.repositoryPath = cmdFlags.Arg(0)
	s := strings.Split(c.repositoryPath, "/")
	if len(s) != 2 {
		c.UI.Error(fmt.Sprintf("failed to parse repository path: %s", c.repositoryPath))
		c.UI.Error(c.Help())
		return 1
	}
	owner := s[0]
	repo := s[1]

	r, err := release.NewGitHubRelease(owner, repo)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	v, err := r.Latest()
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
Usage: tfupdate release latest [options] <REPOSITORY>

Arguments
  REPOSITORY         A path of the the GitHub repository
                     (e.g. terraform-providers/terraform-provider-aws)
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ReleaseLatestCommand) Synopsis() string {
	return "Get the latest release version from GitHub Release"
}
