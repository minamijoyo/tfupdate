package command

import (
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
	flag "github.com/spf13/pflag"
)

// ProviderCommand is a command which update version constraints for provider.
type ProviderCommand struct {
	Meta
	target     string
	path       string
	recursive  bool
	ignorePath string
}

// Run runs the procedure of this command.
func (c *ProviderCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("provider", flag.ContinueOnError)
	cmdFlags.BoolVarP(&c.recursive, "recursive", "r", false, "Check a directory recursively")
	cmdFlags.StringVarP(&c.ignorePath, "ignore-path", "i", "", "A regular expression for path to ignore")

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 2 {
		c.UI.Error(fmt.Sprintf("The command expects 2 arguments, but got %#v", cmdFlags.Args()))
		c.UI.Error(c.Help())
		return 1
	}

	c.target = cmdFlags.Arg(0)
	c.path = cmdFlags.Arg(1)

	option, err := tfupdate.NewOption("provider", c.target, c.recursive, c.ignorePath)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = tfupdate.UpdateFileOrDir(c.Fs, c.path, option)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

// Help returns long-form help text.
func (c *ProviderCommand) Help() string {
	helpText := `
Usage: tfupdate provider [options] <PROVIER_NAME>@<VERSION> <PATH>

Arguments
  PROVIER_NAME       A name of provider (e.g. aws, google, azurerm)
  VERSION            A new version constraint
  PATH               A path of file or directory to update

Options:
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ProviderCommand) Synopsis() string {
	return "Update version constraints for provider"
}
