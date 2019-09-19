package command

import (
	"flag"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
)

// ProviderCommand is a command which update version constraints for provider.
type ProviderCommand struct {
	Meta
	target    string
	path      string
	recursive bool
}

// Run runs the procedure of this command.
func (c *ProviderCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("provider", flag.ContinueOnError)
	cmdFlags.StringVar(&c.target, "v", "", "A new version constraint")
	cmdFlags.StringVar(&c.path, "f", "./", "A path of file or directory to update")
	cmdFlags.BoolVar(&c.recursive, "r", false, "Check a directory recursively")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if len(cmdFlags.Args()) != 0 {
		c.UI.Error("The provider command expects no arguments")
		c.UI.Error(c.Help())
		return 1
	}

	option := tfupdate.NewOption("provider", c.target)
	err := tfupdate.UpdateFileOrDir(c.Fs, c.path, c.recursive, option)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

// Help returns long-form help text.
func (c *ProviderCommand) Help() string {
	helpText := `
Usage: tfupdate provider [options]

Options:
  -v    A new version constraint.
        The valid format is <PROVIER_NAME>@<VERSION>
  -f    A path of file or directory to update (default: ./)
  -r    Check a directory recursively (default: false)
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ProviderCommand) Synopsis() string {
	return "Update version constraints for provider"
}
