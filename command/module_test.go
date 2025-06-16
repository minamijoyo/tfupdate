package command

import (
	"strings"
	"testing"
)

func TestModuleCommand_Help(t *testing.T) {
	c := &ModuleCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "ModuleCommand.Help()")

	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate module",
		"Arguments",
		"MODULE_NAME",
		"PATH",
		"Options:",
		"-v  --version",
		"-r  --recursive",
		"-i  --ignore-path",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(help, expected) {
			t.Errorf("ModuleCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestModuleCommand_Synopsis(t *testing.T) {
	c := &ModuleCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Update version constraints for module", "ModuleCommand.Synopsis()")
}
