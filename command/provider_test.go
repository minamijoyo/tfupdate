package command

import (
	"strings"
	"testing"
)

func TestProviderCommand_Help(t *testing.T) {
	c := &ProviderCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "ProviderCommand.Help()")

	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate provider",
		"Arguments",
		"PROVIDER_NAME",
		"PATH",
		"Options:",
		"-v  --version",
		"-r  --recursive",
		"-i  --ignore-path",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(help, expected) {
			t.Errorf("ProviderCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestProviderCommand_Synopsis(t *testing.T) {
	c := &ProviderCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Update version constraints for provider", "ProviderCommand.Synopsis()")
}
