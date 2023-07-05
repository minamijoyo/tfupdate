package command

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/minamijoyo/tfupdate/tfupdate"
	flag "github.com/spf13/pflag"
)

// LockCommand is a command which update dependency lock files.
type LockCommand struct {
	Meta
	platforms   []string
	path        string
	recursive   bool
	ignorePaths []string
}

// Run runs the procedure of this command.
func (c *LockCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("lock", flag.ContinueOnError)
	cmdFlags.StringArrayVar(&c.platforms, "platform", []string{}, "A target platform for dependecy lock file")
	cmdFlags.BoolVarP(&c.recursive, "recursive", "r", false, "Check a directory recursively")
	cmdFlags.StringArrayVarP(&c.ignorePaths, "ignore-path", "i", []string{}, "A regular expression for path to ignore")

	if err := cmdFlags.Parse(args); err != nil {
		c.UI.Error(fmt.Sprintf("failed to parse arguments: %s", err))
		return 1
	}

	if len(cmdFlags.Args()) != 1 {
		c.UI.Error(fmt.Sprintf("The command expects 1 arguments, but got %d", len(cmdFlags.Args())))
		c.UI.Error(c.Help())
		return 1
	}

	c.path = cmdFlags.Arg(0)

	if filepath.IsAbs(c.path) {
		c.UI.Error("The PATH argument should be a relative path, not an absolute path")
		c.UI.Error(c.Help())
		return 1
	}

	if len(c.platforms) == 0 {
		c.UI.Error("The --platform flag is required")
		c.UI.Error(c.Help())
		return 1
	}

	log.Println("[INFO] Update dependency lock files")
	option, err := tfupdate.NewOption("lock", "", "", c.platforms, c.recursive, c.ignorePaths)
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
func (c *LockCommand) Help() string {
	helpText := `
Usage: tfupdate lock [options] <PATH>

Arguments
  PATH               A relative path of directory to update

Options:
      --platform     Specify a platform to update dependency lock files.
                     At least one or more --platform flags must be specified.
                     Use this option multiple times to include checksums for multiple target systems.
                     Target platform names consist of an operating system and a CPU architecture.
                     (e.g. linux_amd64, darwin_amd64, darwin_arm64)
  -r  --recursive    Check a directory recursively (default: false)
  -i  --ignore-path  A regular expression for path to ignore
                     If you want to ignore multiple directories, set the flag multiple times.
`
	return strings.TrimSpace(helpText)
}

// Synopsis returns one-line help text.
func (c *LockCommand) Synopsis() string {
	return "Update dependency lock files"
}
