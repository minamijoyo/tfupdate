package command

import (
	"strings"
	"testing"
)

func TestReleaseLatestCommand_Help(t *testing.T) {
	c := &ReleaseLatestCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "ReleaseLatestCommand.Help()")

	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate release latest",
		"SOURCE",
		"--source-type",
		"github",
		"gitlab",
		"tfregistryModule",
		"tfregistryProvider",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(help, expected) {
			t.Errorf("ReleaseLatestCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestReleaseLatestCommand_Synopsis(t *testing.T) {
	c := &ReleaseLatestCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Get the latest release version", "ReleaseLatestCommand.Synopsis()")
}
