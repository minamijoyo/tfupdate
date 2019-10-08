package command

import (
	"fmt"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
	flag "github.com/spf13/pflag"
)

// TerraformCommand is a command which update version constraints for terraform.
type TerraformCommand struct {
	Meta
	target      string
	path        string
	recursive   bool
	ignorePaths []string
}

// Run runs the procedure of this command.
func (c *TerraformCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("terraform", flag.ContinueOnError)
	cmdFlags.BoolVarP(&c.recursive, "recursive", "r", false, "Check a directory recursively")
	cmdFlags.StringArrayVarP(&c.ignorePaths, "ignore-path", "i", []string{}, "A regular expression for path to ignore")

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

	option, err := tfupdate.NewOption("terraform", c.target, c.recursive, c.ignorePaths)
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
func (c *TerraformCommand) Help() string {
	helpText := `
Usage: tfupdate terraform [options] <VERSION> <PATH>

Arguments
  VERSION            A new version constraint
  PATH               A path of file or directory to update

Options:
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *TerraformCommand) Synopsis() string {
	return "Update version constraints for terraform"
}
