package command

import (
	"strings"
	"testing"
)

func TestOpenTofuCommand_Help(t *testing.T) {
	c := &OpenTofuCommand{}
	help := c.Help()

	AssertEqual(t, help != "", true, "OpenTofuCommand.Help()")
	// Check that help contains expected sections
	expectedStrings := []string{
		"Usage: tfupdate opentofu",
		"Arguments",
		"PATH",
		"Options:",
		"-v  --version",
		"-r  --recursive",
		"-i  --ignore-path",
	}

	for _, expected := range expectedStrings {
		if !containsString(help, expected) {
			t.Errorf("OpenTofuCommand.Help() missing expected string: %s", expected)
		}
	}
}

func TestOpenTofuCommand_Synopsis(t *testing.T) {
	c := &OpenTofuCommand{}
	synopsis := c.Synopsis()

	AssertEqual(t, synopsis, "Update version constraints for opentofu", "OpenTofuCommand.Synopsis()")
}

// containsString is a helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
