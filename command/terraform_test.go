package command

import (
	"strings"
	"testing"
)

func TestTerraformCommandHelp(t *testing.T) {
	cmd := &TerraformCommand{}
	got := cmd.Help()

	// Check that help text contains expected content
	expectedContents := []string{
		"Usage: tfupdate terraform",
		"Arguments",
		"PATH",
		"Options:",
		"-v  --version",
		"-r  --recursive",
		"-i  --ignore-path",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(got, expected) {
			t.Errorf("TerraformCommand.Help() does not contain expected content: %s\ngot: %s", expected, got)
		}
	}

	// Check that help text is not empty
	if strings.TrimSpace(got) == "" {
		t.Error("TerraformCommand.Help() returns empty string")
	}
}

func TestTerraformCommandSynopsis(t *testing.T) {
	cmd := &TerraformCommand{}
	got := cmd.Synopsis()

	// Check expected synopsis content
	expected := "Update version constraints for terraform"
	if got != expected {
		t.Errorf("TerraformCommand.Synopsis() = %s, want = %s", got, expected)
	}

	// Check that synopsis is not empty
	if strings.TrimSpace(got) == "" {
		t.Error("TerraformCommand.Synopsis() returns empty string")
	}
}
