package command

import (
	"testing"
)

func TestProviderCommandRunParse(t *testing.T) {
	testCases := []struct {
		desc                string
		args                []string
		ok                  bool
		errorMessage        string
		expectedName        string
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
			errorMessage: "The command expects 2 arguments, but got 0",
		},
		{
			desc:         "one argument only",
			args:         []string{"aws"},
			ok:           false,
			errorMessage: "The command expects 2 arguments, but got 1",
		},
		{
			desc:         "too many arguments",
			args:         []string{"aws", "test.tf", "extra"},
			ok:           false,
			errorMessage: "The command expects 2 arguments, but got 3",
		},
		{
			desc:         "invalid flag",
			args:         []string{"--invalid-flag", "aws", "test.tf"},
			ok:           false,
			errorMessage: "failed to parse arguments:",
		},
		// Success cases with argument and flag parsing
		{
			desc:                "default values",
			args:                []string{"aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "version flag short form",
			args:                []string{"-v", "4.0.0", "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "4.0.0",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "version flag long form",
			args:                []string{"--version", "4.0.0", "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "4.0.0",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "recursive flag short form",
			args:                []string{"-r", "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "recursive flag long form",
			args:                []string{"--recursive", "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "ignore-path flag short form",
			args:                []string{"-i", `.*\.backup$`, "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`},
		},
		{
			desc:                "ignore-path flag long form",
			args:                []string{"--ignore-path", `.*\.backup$`, "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`},
		},
		{
			desc:                "multiple ignore-path flags",
			args:                []string{"-i", `.*\.backup$`, "-i", `.*\.tmp$`, "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
		{
			desc:                "namespaced provider",
			args:                []string{"integrations/github", "test.tf"},
			ok:                  true,
			expectedName:        "integrations/github",
			expectedPath:        "test.tf",
			expectedVersion:     "latest",
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "all flags combined",
			args:                []string{"-v", "4.0.0", "-r", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "aws", "test.tf"},
			ok:                  true,
			expectedName:        "aws",
			expectedPath:        "test.tf",
			expectedVersion:     "4.0.0",
			expectedRecursive:   true,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ProviderCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory([]string{"4.0.0", "4.1.0", "4.2.0"}, nil),
				),
			}

			// Create test file if expecting success
			if tc.ok {
				err := WriteTestFile(cmd.Fs, "test.tf", `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}`)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "ProviderCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "ProviderCommand.Run(%v)", tc.args)
				}
			} else {
				// For success cases, check both exit code and parsed values
				if exitCode != 0 {
					t.Errorf("ProviderCommand.Run(%v) returns exit code %d, but want = 0. Error output: %s",
						tc.args, exitCode, ui.GetErrorOutput())
				}

				// Check parsed values
				AssertEqual(t, cmd.name, tc.expectedName, "ProviderCommand.Run(%v) name", tc.args)
				AssertEqual(t, cmd.path, tc.expectedPath, "ProviderCommand.Run(%v) path", tc.args)
				AssertEqual(t, cmd.version, tc.expectedVersion, "ProviderCommand.Run(%v) version", tc.args)
				AssertEqual(t, cmd.recursive, tc.expectedRecursive, "ProviderCommand.Run(%v) recursive", tc.args)
				AssertDeepEqual(t, cmd.ignorePaths, tc.expectedIgnorePaths, "ProviderCommand.Run(%v) ignorePaths", tc.args)
			}
		})
	}
}
