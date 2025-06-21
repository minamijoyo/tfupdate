package command

import (
	"errors"
	"testing"
)

func TestOpenTofuCommandRunParse(t *testing.T) {
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
			args:                []string{"-v", "1.6.0", "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "1.6.0",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "version flag long form",
			args:                []string{"--version", "1.6.0", "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "1.6.0",
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
			args:                []string{"-v", "1.6.0", "-r", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "test.tf"},
			ok:                  true,
			expectedPath:        "test.tf",
			expectedVersion:     "1.6.0",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &OpenTofuCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory([]string{"1.6.0", "1.6.1", "1.6.2"}, nil),
				),
			}

			// Create test file if expecting success
			if tc.ok {
				err := WriteTestFile(cmd.Fs, "test.tf", `terraform {
  required_version = "~> 1.5"
}`)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "OpenTofuCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "OpenTofuCommand.Run(%v)", tc.args)
				}
			} else {
				// For success cases, check both exit code and parsed values
				if exitCode != 0 {
					t.Errorf("OpenTofuCommand.Run(%v) returns exit code %d, but want = 0. Error output: %s",
						tc.args, exitCode, ui.GetErrorOutput())
				}

				// Check parsed values
				AssertEqual(t, cmd.path, tc.expectedPath, "OpenTofuCommand.Run(%v) path", tc.args)
				AssertEqual(t, cmd.version, tc.expectedVersion, "OpenTofuCommand.Run(%v) version", tc.args)
				AssertEqual(t, cmd.recursive, tc.expectedRecursive, "OpenTofuCommand.Run(%v) recursive", tc.args)
				AssertDeepEqual(t, cmd.ignorePaths, tc.expectedIgnorePaths, "OpenTofuCommand.Run(%v) ignorePaths", tc.args)
			}
		})
	}
}

func TestOpenTofuCommandRunUpdate(t *testing.T) {
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
			desc:    "specific version update",
			version: "1.6.0",
			fileContent: `terraform {
  required_version = "~> 1.5"
}`,
			mockVersions:   []string{"1.6.0", "1.6.1"}, // Mock not used for specific version
			mockError:      nil,
			ok:             true,
			expectedUpdate: true,
		},
		{
			desc:    "latest version update",
			version: "latest",
			fileContent: `terraform {
  required_version = "~> 1.5"
}`,
			mockVersions:   []string{"1.6.0", "1.6.1", "1.6.2"},
			mockError:      nil,
			ok:             true,
			expectedUpdate: true,
		},
		// Release error cases
		{
			desc:    "release API error",
			version: "latest",
			fileContent: `terraform {
  required_version = "~> 1.5"
}`,
			mockVersions:     nil,
			mockError:        errors.New("API error"),
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "API error",
		},
		{
			desc:    "no releases available",
			version: "latest",
			fileContent: `terraform {
  required_version = "~> 1.5"
}`,
			mockVersions:     []string{},
			mockError:        nil,
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "no releases found",
		},
		// File processing error cases
		{
			desc:             "invalid terraform file",
			version:          "1.6.0",
			fileContent:      `invalid terraform syntax {`,
			mockVersions:     []string{"1.6.0"},
			mockError:        nil,
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "Unclosed configuration block",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &OpenTofuCommand{
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
				AssertCommandSuccess(t, exitCode, "OpenTofuCommand.Run(%v)", args)

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
				AssertCommandFailure(t, exitCode, "OpenTofuCommand.Run(%v)", args)

				// Check error output contains some error message
				errorOutput := ui.GetErrorOutput()
				if errorOutput == "" {
					t.Error("Expected error output but got empty string")
				}

				// Check specific error message if provided
				if tc.expectedErrorMsg != "" {
					AssertUIError(t, ui, tc.expectedErrorMsg, "OpenTofuCommand.Run(%v)", args)
				}
			}
		})
	}
}
