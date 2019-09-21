package command

import (
	"flag"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
)

// TerraformCommand is a command which update version constraints for terraform.
type TerraformCommand struct {
	Meta
	target     string
	path       string
	recursive  bool
	ignorePath string
}

// Run runs the procedure of this command.
func (c *TerraformCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("terraform", flag.ContinueOnError)
	cmdFlags.StringVar(&c.target, "v", "", "A new version constraint")
	cmdFlags.StringVar(&c.path, "f", "./", "A path of file or directory to update")
	cmdFlags.BoolVar(&c.recursive, "r", false, "Check a directory recursively")
	cmdFlags.StringVar(&c.ignorePath, "ignore-path", "", "A regular expression for path to ignore")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if len(cmdFlags.Args()) != 0 {
		c.UI.Error("The provider command expects no arguments")
		c.UI.Error(c.Help())
		return 1
	}

	if len(c.target) == 0 {
		c.UI.Error("Argument error: -v is required\n")
		c.UI.Error(c.Help())
		return 1
	}

	option, err := tfupdate.NewOption("terraform", c.target, c.recursive, c.ignorePath)
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
Usage: tfupdate terraform [options]

Options:
  -v             A new version constraint
  -f             A path of file or directory to update (default: ./)
  -r             Check a directory recursively (default: false)
  --ignore-path  A regular expression for path to ignore
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *TerraformCommand) Synopsis() string {
	return "Update version constraints for terraform"
}
