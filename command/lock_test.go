package command

import (
	"strings"
	"testing"
)

func TestLockCommand_Help(t *testing.T) {
	c := &LockCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "LockCommand.Help()")

	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate lock",
		"Arguments",
		"PATH",
		"Options:",
		"-r  --recursive",
		"-i  --ignore-path",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(help, expected) {
			t.Errorf("LockCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestLockCommand_Synopsis(t *testing.T) {
	c := &LockCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Update dependency lock files", "LockCommand.Synopsis()")
}
