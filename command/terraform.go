package command

import (
	"flag"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
)

// TerraformCommand is a command which update version constraints for terraform.
type TerraformCommand struct {
	Meta
	path string
}

// Run runs the procedure of this command.
func (c *TerraformCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("terraform", flag.ContinueOnError)
	cmdFlags.StringVar(&c.path, "f", "main.tf", "A path to filename to update")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if len(cmdFlags.Args()) != 1 {
		c.UI.Error("The terraform command expects <VERSION>")
		c.UI.Error(c.Help())
		return 1
	}

	updateType := "terraform"
	target := cmdFlags.Args()[0]
	filename := c.path

	option := tfupdate.NewOption(updateType, target)
	err := tfupdate.UpdateFile(filename, option)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

// Help returns long-form help text.
func (c *TerraformCommand) Help() string {
	helpText := `
Usage: tfupdate terraform [options] <VERSION>

Options:

  -f=path    A path to filename to update
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *TerraformCommand) Synopsis() string {
	return "Update version constraints for terraform"
}
