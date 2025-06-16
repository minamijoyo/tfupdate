package command

import (
	"strings"
	"testing"
)

func TestReleaseCommand_Help(t *testing.T) {
	c := &ReleaseCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "ReleaseCommand.Help()")

	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate release",
		"subcommands for release version information",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(help, expected) {
			t.Errorf("ReleaseCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestReleaseCommand_Synopsis(t *testing.T) {
	c := &ReleaseCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Get release version information", "ReleaseCommand.Synopsis()")
}
