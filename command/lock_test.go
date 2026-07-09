package command

import (
	"testing"
)

func TestLockCommandRunParse(t *testing.T) {
	testCases := []struct {
		desc                string
		args                []string
		ok                  bool
		errorMessage        string
		expectedPath        string
		expectedPlatforms   []string
		expectedRecursive   bool
		expectedIgnorePaths []string
	}{
		// Error cases
		{
			desc:         "no arguments",
			args:         []string{},
			ok:           false,
			errorMessage: "The command expects 1 arguments, but got 0",
		},
		{
			desc:         "too many arguments",
			args:         []string{"path1", "path2"},
			ok:           false,
			errorMessage: "The command expects 1 arguments, but got 2",
		},
		{
			desc:         "invalid flag",
			args:         []string{"--invalid-flag", "."},
			ok:           false,
			errorMessage: "failed to parse arguments:",
		},
		{
			desc:         "absolute path not allowed",
			args:         []string{"--platform", "linux_amd64", "/absolute/path"},
			ok:           false,
			errorMessage: "The PATH argument should be a relative path, not an absolute path",
		},
		{
			desc:         "missing platform flag",
			args:         []string{"."},
			ok:           false,
			errorMessage: "The --platform flag is required",
		},
		// Success cases with argument and flag parsing
		{
			desc:                "single platform",
			args:                []string{"--platform", "linux_amd64", "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "multiple platforms",
			args:                []string{"--platform", "linux_amd64", "--platform", "darwin_amd64", "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64", "darwin_amd64"},
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "recursive flag short form",
			args:                []string{"--platform", "linux_amd64", "-r", "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   true,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "recursive flag long form",
			args:                []string{"--platform", "linux_amd64", "--recursive", "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   true,
			expectedIgnorePaths: []string{},
		},
		{
			desc:                "ignore-path flag short form",
			args:                []string{"--platform", "linux_amd64", "-i", `.*\.backup$`, "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`},
		},
		{
			desc:                "ignore-path flag long form",
			args:                []string{"--platform", "linux_amd64", "--ignore-path", `.*\.backup$`, "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`},
		},
		{
			desc:                "multiple ignore-path flags",
			args:                []string{"--platform", "linux_amd64", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   false,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
		{
			desc:                "all flags combined",
			args:                []string{"--platform", "linux_amd64", "--platform", "darwin_amd64", "-r", "-i", `.*\.backup$`, "-i", `.*\.tmp$`, "."},
			ok:                  true,
			expectedPath:        ".",
			expectedPlatforms:   []string{"linux_amd64", "darwin_amd64"},
			expectedRecursive:   true,
			expectedIgnorePaths: []string{`.*\.backup$`, `.*\.tmp$`},
		},
		{
			desc:                "relative subdirectory path",
			args:                []string{"--platform", "linux_amd64", "subdir"},
			ok:                  true,
			expectedPath:        "subdir",
			expectedPlatforms:   []string{"linux_amd64"},
			expectedRecursive:   false,
			expectedIgnorePaths: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ui := NewMockUI()
			cmd := &LockCommand{
				Meta: NewTestMetaWithUI(ui),
			}

			// Note: Parse tests focus on argument validation, not actual file operations

			exitCode := cmd.Run(tc.args)

			if !tc.ok {
				AssertCommandFailure(t, exitCode, "LockCommand.Run(%v)", tc.args)
				if tc.errorMessage != "" {
					AssertUIError(t, ui, tc.errorMessage, "LockCommand.Run(%v)", tc.args)
				}
			} else {
				// For parse test success cases, we only verify argument parsing succeeded
				// The actual exit code may be non-zero due to missing lock files, which is expected
				// Check parsed values
				AssertEqual(t, cmd.path, tc.expectedPath, "LockCommand.Run(%v) path", tc.args)
				AssertDeepEqual(t, cmd.platforms, tc.expectedPlatforms, "LockCommand.Run(%v) platforms", tc.args)
				AssertEqual(t, cmd.recursive, tc.expectedRecursive, "LockCommand.Run(%v) recursive", tc.args)
				AssertDeepEqual(t, cmd.ignorePaths, tc.expectedIgnorePaths, "LockCommand.Run(%v) ignorePaths", tc.args)
			}
		})
	}
}

// TestLockCommandRunUpdate would test the actual lock file update functionality,
// but it requires mocking the Terraform Registry API to prevent external HTTP calls
// during unit tests. The lock file update logic itself is tested separately in the
// tfupdate package. For command-level testing, TestLockCommandRunParse covers the
// argument parsing functionality adequately.
