package command

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minamijoyo/tfupdate/release"
	"github.com/minamijoyo/tfupdate/tfupdate"
	flag "github.com/spf13/pflag"
)

// ProviderCommand is a command which update version constraints for provider.
type ProviderCommand struct {
	Meta
	name        string
	version     string
	path        string
	recursive   bool
	ignorePaths []string
}

// Run runs the procedure of this command.
func (c *ProviderCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("provider", flag.ContinueOnError)
	cmdFlags.StringVarP(&c.version, "version", "v", "latest", "A new version constraint")
	cmdFlags.BoolVarP(&c.recursive, "recursive", "r", false, "Check a directory recursively")
	cmdFlags.StringArrayVarP(&c.ignorePaths, "ignore-path", "i", []string{}, "A regular expression for path to ignore")

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
	if v == "latest" {
		source := ""
		if strings.Contains(c.name, "/") {
			namespace, name, _ := strings.Cut(c.name, "/")
			source = fmt.Sprintf("%s/terraform-provider-%s", namespace, name)
		} else {
			source = fmt.Sprintf("hashicorp/terraform-provider-%s", c.name)
		}
		r, err := newRelease("github", source)
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

	log.Printf("[INFO] Update provider %s to %s", c.name, v)
	option, err := tfupdate.NewOption("provider", c.name, v, []string{}, c.recursive, c.ignorePaths)
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
func (c *ProviderCommand) Help() string {
	helpText := `
Usage: tfupdate provider [options] <PROVIDER_NAME> <PATH>

Arguments
  PROVIDER_NAME      A name of provider (e.g. aws or integrations/github)
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
func (c *ProviderCommand) Synopsis() string {
	return "Update version constraints for provider"
}
