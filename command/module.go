package command

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
	flag "github.com/spf13/pflag"
)

// ModuleCommand is a command which update version constraints for module.
type ModuleCommand struct {
	Meta
	name            string
	version         string
	path            string
	recursive       bool
	ignorePaths     []string
	sourceMatchType string
}

// Run runs the procedure of this command.
func (c *ModuleCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("module", flag.ContinueOnError)
	cmdFlags.StringVarP(&c.version, "version", "v", "", "A new version constraint")
	cmdFlags.BoolVarP(&c.recursive, "recursive", "r", false, "Check a directory recursively")
	cmdFlags.StringArrayVarP(&c.ignorePaths, "ignore-path", "i", []string{}, "A regular expression for path to ignore")
	cmdFlags.StringVar(&c.sourceMatchType, "source-match-type", "full", "Define how to match module source URLs. Valid values are \"full\" or \"regex\".")

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 2 {
		c.UI.Error(fmt.Sprintf("The command expects 2 arguments, but got %d", len(cmdFlags.Args())))
		c.UI.Error(c.Help())
		return 1
	}

	c.name = cmdFlags.Arg(0)
	c.path = cmdFlags.Arg(1)

	v := c.version
	if len(v) == 0 {
		// For modules, automatic latest version resolution is not simple.
		// To implement, we will probably need to get information from the Terraform Registry.
		c.UI.Error("A new version constraint is required. Automatic latest version resolution is not currently supported for modules.")
		return 1
	}

	log.Printf("[INFO] Update module %s to %s", c.name, v)
	option, err := tfupdate.NewOption("module", c.name, v, []string{}, c.recursive, c.ignorePaths, c.sourceMatchType)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	gc, err := tfupdate.NewGlobalContext(c.Fs, option)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = tfupdate.UpdateFileOrDir(context.Background(), gc, c.path)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

// Help returns long-form help text.
func (c *ModuleCommand) Help() string {
	helpText := `
Usage: tfupdate module [options] <MODULE_NAME> <PATH>

Arguments
  MODULE_NAME        A name of module or a regular expression in RE2 syntax
                     e.g.
                       terraform-aws-modules/vpc/aws
                       git::https://example.com/vpc.git
                       git::https://example.com/.+
  PATH               A path of file or directory to update

Options:
  -v  --version       A new version constraint (required)
                      Automatic latest version resolution is not currently supported for modules.
  -r  --recursive     Check a directory recursively (default: false)
  -i  --ignore-path   A regular expression for path to ignore
                      If you want to ignore multiple directories, set the flag multiple times.
  --source-match-type Define how to match MODULE_NAME to the module source URLs. Valid values are "full" or "regex". (default: full)
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *ModuleCommand) Synopsis() string {
	return "Update version constraints for module"
}
