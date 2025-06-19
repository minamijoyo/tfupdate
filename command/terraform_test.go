package command

import (
	"errors"
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

func TestTerraformCommandRunParse(t *testing.T) {
	testCases := []struct {
		desc                string
		args                []string
		ok                  bool
		errorMessage        string
		expectedPath        string
		expectedVersion     string
		expectedRecursive   bool
		expectedIgnorePaths []string
	}{
		// Error cases
		{
			desc:         "no arguments",
			args:         []string{},
			ok:           false,
			errorMessage: "The command expects 1 argument, but got 0",
		},
		{
			desc:         "too many arguments",
			args:         []string{"path1", "path2"},
			ok:           false,
			errorMessage: "The command expects 1 argument, but got 2",
		},
		{
			desc:         "invalid flag",
			args:         []string{"--invalid-flag", "test.tf"},
			ok:           false,
			errorMessage: "failed to parse arguments:",
		},
		// Success cases with argument and flag parsing
		{
			desc:                "default values",
			args:                []string{"test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "version flag short form",
			args:                []string{"-v", "1.0.0", "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "1.0.0",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "version flag long form",
			args:                []string{"--version", "1.0.0", "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "1.0.0",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "recursive flag short form",
			args:                []string{"-r", "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "recursive flag long form",
			args:                []string{"--recursive", "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "ignore-path flag short form",
			args:                []string{"-i", `.*\.backup$`, "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`},
		},
		{
			desc:                "ignore-path flag long form",
			args:                []string{"--ignore-path", `.*\.backup$`, "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`},
		},
		{
			desc:                "multiple ignore-path flags",
			args:                []string{"-i", `.*\.backup$`, "-i", `.*\.tmp$`, "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
		{
			desc:                "all flags combined",
			args:                []string{"-v", "1.0.0", "-r", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "1.0.0",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &TerraformCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory([]string{"1.0.0", "1.1.0", "1.2.0"}, nil),
				),
			}

			// Create test file if expecting success
			if tc.ok {
				err := WriteTestFile(cmd.Fs, "test.tf", `terraform {
  required_version = "~> 0.12"
}`)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "TerraformCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "TerraformCommand.Run(%v)", tc.args)
				}
			} else {
				// For success cases, check both exit code and parsed values
				if exitCode != 0 {
					t.Errorf("TerraformCommand.Run(%v) returns exit code %d, but want = 0. Error output: %s",
						tc.args, exitCode, ui.GetErrorOutput())
				}

				// Check parsed values
				AssertEqual(t, cmd.path, tc.expectedPath, "TerraformCommand.Run(%v) path", tc.args)
				AssertEqual(t, cmd.version, tc.expectedVersion, "TerraformCommand.Run(%v) version", tc.args)
				AssertEqual(t, cmd.recursive, tc.expectedRecursive, "TerraformCommand.Run(%v) recursive", tc.args)
				AssertDeepEqual(t, cmd.ignorePaths, tc.expectedIgnorePaths, "TerraformCommand.Run(%v) ignorePaths", tc.args)
			}
		})
	}
}

func TestTerraformCommandRunUpdate(t *testing.T) {
	testCases := []struct {
		desc             string
		version          string
		fileContent      string
		mockVersions     []string
		mockError        error
		ok               bool
		expectedUpdate   bool
		expectedErrorMsg string
	}{
		// Success cases
		{
			desc:           "specific version update",
			version:        "1.0.0",
			fileContent:    `terraform { required_version = "~> 0.12" }`,
			mockVersions:   []string{"1.0.0", "1.1.0"}, // Mock not used for specific version
			mockError:      nil,
			ok:             true,
			expectedUpdate: true,
		},
		{
			desc:           "latest version update",
			version:        "latest",
			fileContent:    `terraform { required_version = "~> 0.12" }`,
			mockVersions:   []string{"1.0.0", "1.1.0", "1.2.0"},
			mockError:      nil,
			ok:             true,
			expectedUpdate: true,
		},
		// Release error cases
		{
			desc:             "release API error",
			version:          "latest",
			fileContent:      `terraform { required_version = "~> 0.12" }`,
			mockVersions:     nil,
			mockError:        errors.New("API error"),
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "API error",
		},
		{
			desc:             "no releases available",
			version:          "latest",
			fileContent:      `terraform { required_version = "~> 0.12" }`,
			mockVersions:     []string{},
			mockError:        nil,
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "no releases found",
		},
		// File processing error cases
		{
			desc:             "invalid terraform file",
			version:          "1.0.0",
			fileContent:      `invalid terraform syntax {`,
			mockVersions:     []string{"1.0.0"},
			mockError:        nil,
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "Unclosed configuration block",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &TerraformCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory(tc.mockVersions, tc.mockError),
				),
			}

			// Create test file
			err := WriteTestFile(cmd.Fs, "test.tf", tc.fileContent)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			args := []string{"-v", tc.version, "test.tf"}
			exitCode := cmd.Run(args)

			if tc.ok {
				AssertCommandSuccess(t, exitCode, "TerraformCommand.Run(%v)", args)

				if tc.expectedUpdate {
					// Check that file was updated
					updatedContent, err := ReadTestFile(cmd.Fs, "test.tf")
					AssertNoError(t, err, "Reading updated file")

					// The content should be different from original
					if updatedContent == tc.fileContent {
						t.Errorf("File content was not updated. Expected change but got same content: %s", updatedContent)
					}
				}
			} else {
				AssertCommandFailure(t, exitCode, "TerraformCommand.Run(%v)", args)

				// Check error output contains some error message
				errorOutput := ui.GetErrorOutput()
				if errorOutput == "" {
					t.Error("Expected error output but got empty string")
				}

				// Check specific error message if provided
				if tc.expectedErrorMsg != "" {
					AssertUIError(t, ui, tc.expectedErrorMsg, "TerraformCommand.Run(%v)", args)
				}
			}
		})
	}
}
