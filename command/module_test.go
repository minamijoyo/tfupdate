package command

import (
	"testing"
)

func TestModuleCommandRunParse(t *testing.T) {
	testCases := []struct {
		desc                    string
		args                    []string
		ok                      bool
		errorMessage            string
		expectedName            string
		expectedPath            string
		expectedVersion         string
		expectedRecursive       bool
		expectedIgnorePaths     []string
		expectedSourceMatchType string
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
			args:         []string{"terraform-aws-modules/vpc/aws"},
			ok:           false,
			errorMessage: "The command expects 2 arguments, but got 1",
		},
		{
			desc:         "too many arguments",
			args:         []string{"terraform-aws-modules/vpc/aws", "test.tf", "extra"},
			ok:           false,
			errorMessage: "The command expects 2 arguments, but got 3",
		},
		{
			desc:         "invalid flag",
			args:         []string{"--invalid-flag", "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:           false,
			errorMessage: "failed to parse arguments:",
		},
		{
			desc:         "missing version flag",
			args:         []string{"terraform-aws-modules/vpc/aws", "test.tf"},
			ok:           false,
			errorMessage: "A new version constraint is required",
		},
		// Success cases with argument and flag parsing
		{
			desc:                    "required version provided",
			args:                    []string{"-v", "3.0.0", "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       false,
			expectedIgnorePaths:     []string{},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "version flag long form",
			args:                    []string{"--version", "3.0.0", "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       false,
			expectedIgnorePaths:     []string{},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "recursive flag short form",
			args:                    []string{"-v", "3.0.0", "-r", "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       true,
			expectedIgnorePaths:     []string{},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "recursive flag long form",
			args:                    []string{"-v", "3.0.0", "--recursive", "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       true,
			expectedIgnorePaths:     []string{},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "ignore-path flag short form",
			args:                    []string{"-v", "3.0.0", "-i", `.*\.backup$`, "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       false,
			expectedIgnorePaths:     []string{`.*\.backup$`},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "ignore-path flag long form",
			args:                    []string{"-v", "3.0.0", "--ignore-path", `.*\.backup$`, "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       false,
			expectedIgnorePaths:     []string{`.*\.backup$`},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "multiple ignore-path flags",
			args:                    []string{"-v", "3.0.0", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       false,
			expectedIgnorePaths:     []string{`.*\.backup$`, `.*\.tmp$`},
			expectedSourceMatchType: "full",
		},
		{
			desc:                    "source-match-type regex",
			args:                    []string{"-v", "3.0.0", "--source-match-type", "regex", "git::https://example\\.com/.+", "test.tf"},
			ok:                      true,
			expectedName:            "git::https://example\\.com/.+",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       false,
			expectedIgnorePaths:     []string{},
			expectedSourceMatchType: "regex",
		},
		{
			desc:                    "all flags combined",
			args:                    []string{"-v", "3.0.0", "-r", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "--source-match-type", "regex", "terraform-aws-modules/vpc/aws", "test.tf"},
			ok:                      true,
			expectedName:            "terraform-aws-modules/vpc/aws",
			expectedPath:            "test.tf",
			expectedVersion:         "3.0.0",
			expectedRecursive:       true,
			expectedIgnorePaths:     []string{`.*\.backup$`, `.*\.tmp$`},
			expectedSourceMatchType: "regex",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ModuleCommand{
				Meta: NewTestMetaWithUI(ui),
			}

			// Create test file if expecting success
			if tc.ok {
				err := WriteTestFile(cmd.Fs, "test.tf", `module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 2.0"
}`)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "ModuleCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "ModuleCommand.Run(%v)", tc.args)
				}
			} else {
				// For success cases, check both exit code and parsed values
				if exitCode != 0 {
					t.Errorf("ModuleCommand.Run(%v) returns exit code %d, but want = 0. Error output: %s",
						tc.args, exitCode, ui.GetErrorOutput())
				}

				// Check parsed values
				AssertEqual(t, cmd.name, tc.expectedName, "ModuleCommand.Run(%v) name", tc.args)
				AssertEqual(t, cmd.path, tc.expectedPath, "ModuleCommand.Run(%v) path", tc.args)
				AssertEqual(t, cmd.version, tc.expectedVersion, "ModuleCommand.Run(%v) version", tc.args)
				AssertEqual(t, cmd.recursive, tc.expectedRecursive, "ModuleCommand.Run(%v) recursive", tc.args)
				AssertDeepEqual(t, cmd.ignorePaths, tc.expectedIgnorePaths, "ModuleCommand.Run(%v) ignorePaths", tc.args)
				AssertEqual(t, cmd.sourceMatchType, tc.expectedSourceMatchType, "ModuleCommand.Run(%v) sourceMatchType", tc.args)
			}
		})
	}
}

func TestModuleCommandRunUpdate(t *testing.T) {
	testCases := []struct {
		desc             string
		moduleName       string
		version          string
		fileContent      string
		ok               bool
		expectedUpdate   bool
		expectedErrorMsg string
	}{
		// Success cases
		{
			desc:       "module version update",
			moduleName: "terraform-aws-modules/vpc/aws",
			version:    "3.0.0",
			fileContent: `module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 2.0"
}`,
			ok:             true,
			expectedUpdate: true,
		},
		{
			desc:       "git module version update",
			moduleName: "git::https://example.com/vpc.git",
			version:    "v1.0.0",
			fileContent: `module "vpc" {
  source = "git::https://example.com/vpc.git?ref=v0.1.0"
}`,
			ok:             true,
			expectedUpdate: true,
		},
		// File processing error cases
		{
			desc:             "invalid terraform file",
			moduleName:       "terraform-aws-modules/vpc/aws",
			version:          "3.0.0",
			fileContent:      `invalid terraform syntax {`,
			ok:               false,
			expectedUpdate:   false,
			expectedErrorMsg: "Unclosed configuration block",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ModuleCommand{
				Meta: NewTestMetaWithUI(ui),
			}

			// Create test file
			err := WriteTestFile(cmd.Fs, "test.tf", tc.fileContent)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			args := []string{"-v", tc.version, tc.moduleName, "test.tf"}
			exitCode := cmd.Run(args)

			if tc.ok {
				AssertCommandSuccess(t, exitCode, "ModuleCommand.Run(%v)", args)

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
				AssertCommandFailure(t, exitCode, "ModuleCommand.Run(%v)", args)

				// Check error output contains some error message
				errorOutput := ui.GetErrorOutput()
				if errorOutput == "" {
					t.Error("Expected error output but got empty string")
				}

				// Check specific error message if provided
				if tc.expectedErrorMsg != "" {
					AssertUIError(t, ui, tc.expectedErrorMsg, "ModuleCommand.Run(%v)", args)
				}
			}
		})
	}
}
