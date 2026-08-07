package command

import (
	"errors"
	"strings"
	"testing"
)

func TestReleaseLatestCommandRunParse(t *testing.T) {
	testCases := []struct {
		desc               string
		args               []string
		ok                 bool
		errorMessage       string
		expectedSource     string
		expectedSourceType string
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
			desc:               "default source type (github)",
			args:               []string{"hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
		},
		{
			desc:               "github source type short form",
			args:               []string{"-s", "github", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
		},
		{
			desc:               "github source type long form",
			args:               []string{"--source-type", "github", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "github",
		},
		{
			desc:               "gitlab source type",
			args:               []string{"-s", "gitlab", "hashicorp/terraform"},
			ok:                 true,
			expectedSource:     "hashicorp/terraform",
			expectedSourceType: "gitlab",
		},
		{
			desc:               "tfregistryModule source type",
			args:               []string{"-s", "tfregistryModule", "terraform-aws-modules/vpc/aws"},
			ok:                 true,
			expectedSource:     "terraform-aws-modules/vpc/aws",
			expectedSourceType: "tfregistryModule",
		},
		{
			desc:               "tfregistryProvider source type",
			args:               []string{"-s", "tfregistryProvider", "hashicorp/aws"},
			ok:                 true,
			expectedSource:     "hashicorp/aws",
			expectedSourceType: "tfregistryProvider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ReleaseLatestCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory([]string{"1.0.0", "1.1.0", "1.2.0"}, nil),
				),
			}

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "ReleaseLatestCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "ReleaseLatestCommand.Run(%v)", tc.args)
				}
			} else {
				// For success cases, check both exit code and parsed values
				if exitCode != 0 {
					t.Errorf("ReleaseLatestCommand.Run(%v) returns exit code %d, but want = 0. Error output: %s",
						tc.args, exitCode, ui.GetErrorOutput())
				}

				// Check parsed values
				AssertEqual(t, cmd.source, tc.expectedSource, "ReleaseLatestCommand.Run(%v) source", tc.args)
				AssertEqual(t, cmd.sourceType, tc.expectedSourceType, "ReleaseLatestCommand.Run(%v) sourceType", tc.args)
			}
		})
	}
}

func TestReleaseLatestCommandRunLatest(t *testing.T) {
	testCases := []struct {
		desc             string
		sourceType       string
		source           string
		mockVersions     []string
		mockError        error
		ok               bool
		expectedOutput   string
		expectedErrorMsg string
	}{
		// Success cases
		{
			desc:           "github latest version",
			sourceType:     "github",
			source:         "hashicorp/terraform",
			mockVersions:   []string{"1.0.0", "1.1.0", "1.2.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: "1.2.0",
		},
		{
			desc:           "gitlab latest version",
			sourceType:     "gitlab",
			source:         "hashicorp/terraform",
			mockVersions:   []string{"2.0.0", "2.1.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: "2.1.0",
		},
		{
			desc:           "tfregistryModule latest version",
			sourceType:     "tfregistryModule",
			source:         "terraform-aws-modules/vpc/aws",
			mockVersions:   []string{"3.0.0", "3.1.0", "3.2.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: "3.2.0",
		},
		{
			desc:           "tfregistryProvider latest version",
			sourceType:     "tfregistryProvider",
			source:         "hashicorp/aws",
			mockVersions:   []string{"4.0.0", "4.1.0"},
			mockError:      nil,
			ok:             true,
			expectedOutput: "4.1.0",
		},
		// Release error cases
		{
			desc:             "release API error",
			sourceType:       "github",
			source:           "hashicorp/terraform",
			mockVersions:     nil,
			mockError:        errors.New("API error"),
			ok:               false,
			expectedErrorMsg: "API error",
		},
		{
			desc:             "no releases available",
			sourceType:       "github",
			source:           "hashicorp/terraform",
			mockVersions:     []string{},
			mockError:        nil,
			ok:               false,
			expectedErrorMsg: "no releases found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &ReleaseLatestCommand{
				Meta: NewTestMetaWithUIAndReleaseFactory(
					ui,
					NewMockReleaseFactory(tc.mockVersions, tc.mockError),
				),
			}

			args := []string{"-s", tc.sourceType, tc.source}
			exitCode := cmd.Run(args)

			if tc.ok {
				AssertCommandSuccess(t, exitCode, "ReleaseLatestCommand.Run(%v)", args)

				// Check output contains expected version
				output := ui.GetOutput()
				if !strings.Contains(output, tc.expectedOutput) {
					t.Errorf("ReleaseLatestCommand.Run(%v) output = %s, want to contain = %s",
						args, output, tc.expectedOutput)
				}
			} else {
				AssertCommandFailure(t, exitCode, "ReleaseLatestCommand.Run(%v)", args)

				// Check error output contains some error message
				errorOutput := ui.GetErrorOutput()
				if errorOutput == "" {
					t.Error("Expected error output but got empty string")
				}

				// Check specific error message if provided
				if tc.expectedErrorMsg != "" {
					AssertUIError(t, ui, tc.expectedErrorMsg, "ReleaseLatestCommand.Run(%v)", args)
				}
			}
		})
	}
}
