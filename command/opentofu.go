package command

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minamijoyo/tfupdate/release"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"github.com/minamijoyo/tfupdate/tfupdate"
	flag "github.com/spf13/pflag"
)

// OpenTofuCommand is a command which update version constraints for OpenTofu.
type OpenTofuCommand struct {
	Meta
	version     string
	path        string
	recursive   bool
	ignorePaths []string
}

// Run runs the procedure of this command.
func (c *OpenTofuCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("opentofu", flag.ContinueOnError)
	cmdFlags.StringVarP(&c.version, "version", "v", "latest", "A new version constraint")
	cmdFlags.BoolVarP(&c.recursive, "recursive", "r", false, "Check a directory recursively")
	cmdFlags.StringArrayVarP(&c.ignorePaths, "ignore-path", "i", []string{}, "A regular expression for path to ignore")

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 1 {
		c.UI.Error(fmt.Sprintf("The command expects 1 argument, but got %d", len(cmdFlags.Args())))
		c.UI.Error(c.Help())
		return 1
	}

	c.path = cmdFlags.Arg(0)

	v := c.version
	if v == "latest" {
		r, err := c.NewRelease("github", "opentofu/opentofu")
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}

		v, err = release.Latest(context.Background(), r)
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	log.Printf("[INFO] Update opentofu to %s", v)
	option, err := tfupdate.NewOption("opentofu", "", v, []string{}, c.recursive, c.ignorePaths, "", tfregistry.Config{})
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
func (c *OpenTofuCommand) Help() string {
	helpText := `
Usage: tfupdate opentofu [options] <PATH>

Arguments
  PATH               A path of file or directory to update

Options:
  -v  --version      A new version constraint (default: latest)
                     If the version is omitted, the latest version is automatically checked and set.
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *OpenTofuCommand) Synopsis() string {
	return "Update version constraints for opentofu"
}
