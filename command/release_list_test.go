package command

import (
	"strings"
	"testing"
)

func TestReleaseListCommand_Help(t *testing.T) {
	c := &ReleaseListCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "ReleaseListCommand.Help()")

	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate release list",
		"SOURCE",
		"--source-type",
		"--max-length",
		"--pre-release",
		"github",
		"gitlab",
		"tfregistryModule",
		"tfregistryProvider",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(help, expected) {
			t.Errorf("ReleaseListCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestReleaseListCommand_Synopsis(t *testing.T) {
	c := &ReleaseListCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Get a list of release versions", "ReleaseListCommand.Synopsis()")
}
