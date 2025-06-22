package command

import (
	"testing"
)

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
