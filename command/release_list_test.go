package command

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestReleaseListCommandRunParse(t *testing.T) {
	testCases := []struct {
		desc               string
		args               []string
		ok                 bool
		errorMessage       string
		expectedSource     string
		expectedSourceType string
		expectedMaxLength  int
		expectedPreRelease bool
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
			args:         []string{"source1", "source2"},
			ok:           false,
			errorMessage: "The command expects 1 argument, but got 2",
		},
		{
			desc:         "invalid flag",
			args:         []string{"--invalid-flag", "hashicorp/terraform"},
			ok:           false,
			errorMessage: "failed to parse arguments:",
		},
		// Success cases with argument and flag parsing
		{
			desc:               "default values",
			args:               []string{"hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
			expectedMaxLength:  10,
			expectedPreRelease: false,
		},
		{
			desc:               "max-length flag short form",
			args:               []string{"-n", "5", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
			expectedMaxLength:  5,
			expectedPreRelease: false,
		},
		{
			desc:               "max-length flag long form",
			args:               []string{"--max-length", "20", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
			expectedMaxLength:  20,
			expectedPreRelease: false,
		},
		{
			desc:               "pre-release flag",
			args:               []string{"--pre-release", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
			expectedMaxLength:  10,
			expectedPreRelease: true,
		},
		{
			desc:               "source-type flag short form",
			args:               []string{"-s", "gitlab", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "gitlab",
			expectedMaxLength:  10,
			expectedPreRelease: false,
		},
		{
			desc:               "all flags combined",
			args:               []string{"-s", "tfregistryModule", "-n", "15", "--pre-release", "terraform-aws-modules/vpc/aws"},
			ok:                 true,
			expectedSource:     "terraform-aws-modules/vpc/aws",
			expectedSourceType: "tfregistryModule",
			expectedMaxLength:  15,
			expectedPreRelease: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ReleaseListCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory([]string{"1.0.0", "1.1.0", "1.2.0"}, nil),
				),
			}

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "ReleaseListCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "ReleaseListCommand.Run(%v)", tc.args)
				}
			} else {
				// For success cases, check both exit code and parsed values
				if exitCode != 0 {
					t.Errorf("ReleaseListCommand.Run(%v) returns exit code %d, but want = 0. Error output: %s",
						tc.args, exitCode, ui.GetErrorOutput())
				}

				// Check parsed values
				AssertEqual(t, cmd.source, tc.expectedSource, "ReleaseListCommand.Run(%v) source", tc.args)
				AssertEqual(t, cmd.sourceType, tc.expectedSourceType, "ReleaseListCommand.Run(%v) sourceType", tc.args)
				AssertEqual(t, cmd.maxLength, tc.expectedMaxLength, "ReleaseListCommand.Run(%v) maxLength", tc.args)
				AssertEqual(t, cmd.preRelease, tc.expectedPreRelease, "ReleaseListCommand.Run(%v) preRelease", tc.args)
			}
		})
	}
}

func TestReleaseListCommandRunList(t *testing.T) {
	testCases := []struct {
		desc             string
		sourceType       string
		source           string
		maxLength        int
		preRelease       bool
		mockVersions     []string
		mockError        error
		ok               bool
		expectedOutput   []string
		expectedErrorMsg string
	}{
		// Success cases
		{
			desc:           "github version list",
			sourceType:     "github",
			source:         "hashicorp/terraform",
			maxLength:      10,
			preRelease:     false,
			mockVersions:   []string{"1.0.0", "1.1.0", "1.2.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: []string{"1.0.0", "1.1.0", "1.2.0"},
		},
		{
			desc:           "gitlab version list with max length",
			sourceType:     "gitlab",
			source:         "hashicorp/terraform",
			maxLength:      2,
			preRelease:     false,
			mockVersions:   []string{"2.0.0", "2.1.0", "2.2.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: []string{"2.1.0", "2.2.0"}, // Latest 2 versions
		},
		{
			desc:           "tfregistryModule version list with pre-release",
			sourceType:     "tfregistryModule",
			source:         "terraform-aws-modules/vpc/aws",
			maxLength:      5,
			preRelease:     true,
			mockVersions:   []string{"3.0.0", "3.1.0-beta", "3.2.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: []string{"3.0.0", "3.1.0-beta", "3.2.0"},
		},
		// Release error cases
		{
			desc:             "release API error",
			sourceType:       "github",
			source:           "hashicorp/terraform",
			maxLength:        10,
			preRelease:       false,
			mockVersions:     nil,
			mockError:        errors.New("API error"),
			ok:               false,
			expectedErrorMsg: "API error",
		},
		{
			desc:           "no releases available",
			sourceType:     "github",
			source:         "hashicorp/terraform",
			maxLength:      10,
			preRelease:     false,
			mockVersions:   []string{},
			mockError:      nil,
			ok:             true,       // Empty list is not an error for release list command
			expectedOutput: []string{}, // Empty output
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ReleaseListCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory(tc.mockVersions, tc.mockError),
				),
			}

			args := []string{"-s", tc.sourceType, "-n", strconv.Itoa(tc.maxLength)}
			if tc.preRelease {
				args = append(args, "--pre-release")
			}
			args = append(args, tc.source)

			exitCode := cmd.Run(args)

			if tc.ok {
				AssertCommandSuccess(t, exitCode, "ReleaseListCommand.Run(%v)", args)

				// Check output contains expected versions
				output := ui.GetOutput()
				if len(tc.expectedOutput) == 0 {
					// For empty output, check that output is empty or only contains whitespace
					if strings.TrimSpace(output) != "" {
						t.Errorf("ReleaseListCommand.Run(%v) output = %s, want empty output", args, output)
					}
				} else {
					for _, expectedVersion := range tc.expectedOutput {
						if !strings.Contains(output, expectedVersion) {
							t.Errorf("ReleaseListCommand.Run(%v) output = %s, want to contain = %s",
								args, output, expectedVersion)
						}
					}
				}
			} else {
				AssertCommandFailure(t, exitCode, "ReleaseListCommand.Run(%v)", args)

				// Check error output contains some error message
				errorOutput := ui.GetErrorOutput()
				if errorOutput == "" {
					t.Error("Expected error output but got empty string")
				}

				// Check specific error message if provided
				if tc.expectedErrorMsg != "" {
					AssertUIError(t, ui, tc.expectedErrorMsg, "ReleaseListCommand.Run(%v)", args)
				}
			}
		})
	}
}
