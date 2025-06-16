package command

import (
	"bufio"
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/cli"
	"github.com/spf13/afero"
)

// MockUI is a mock implementation of cli.Ui for testing.
type MockUI struct {
	ErrorWriter  bytes.Buffer
	OutputWriter bytes.Buffer
	InputReader  *bufio.Reader
}

// NewMockUI creates a new MockUI instance for testing.
func NewMockUI() *MockUI {
	return &MockUI{
		InputReader: bufio.NewReader(strings.NewReader("")),
	}
}

// Ask reads a line of input from the input reader.
func (u *MockUI) Ask(_ string) (string, error) {
	return u.InputReader.ReadString('\n')
}

// AskSecret reads a line of input from the input reader without echoing.
func (u *MockUI) AskSecret(_ string) (string, error) {
	return u.InputReader.ReadString('\n')
}

// Error writes to the error writer.
func (u *MockUI) Error(message string) {
	u.ErrorWriter.WriteString(message)
}

// Info writes to the output writer.
func (u *MockUI) Info(message string) {
	u.OutputWriter.WriteString(message)
}

// Output writes to the output writer.
func (u *MockUI) Output(message string) {
	u.OutputWriter.WriteString(message)
}

// Warn writes to the error writer.
func (u *MockUI) Warn(message string) {
	u.ErrorWriter.WriteString(message)
}

// GetErrorOutput returns the content written to the error writer.
func (u *MockUI) GetErrorOutput() string {
	return u.ErrorWriter.String()
}

// GetOutput returns the content written to the output writer.
func (u *MockUI) GetOutput() string {
	return u.OutputWriter.String()
}

// SetInput sets the input for Ask and AskSecret methods.
func (u *MockUI) SetInput(input string) {
	u.InputReader = bufio.NewReader(strings.NewReader(input))
}

// Verify MockUI implements cli.Ui interface.
var _ cli.Ui = (*MockUI)(nil)

// NewTestMeta creates a Meta instance with mock UI and in-memory file system for testing.
func NewTestMeta() Meta {
	return Meta{
		UI: NewMockUI(),
		Fs: afero.NewMemMapFs(),
	}
}

// NewTestMetaWithUI creates a Meta instance with provided UI and in-memory file system for testing.
func NewTestMetaWithUI(ui cli.Ui) Meta {
	return Meta{
		UI: ui,
		Fs: afero.NewMemMapFs(),
	}
}

// NewTestMetaWithFs creates a Meta instance with mock UI and provided file system for testing.
func NewTestMetaWithFs(fs afero.Fs) Meta {
	return Meta{
		UI: NewMockUI(),
		Fs: fs,
	}
}

// WriteTestFile is a helper function to write test files to the file system.
func WriteTestFile(fs afero.Fs, filename string, content string) error {
	return afero.WriteFile(fs, filename, []byte(content), 0644)
}

// ReadTestFile is a helper function to read test files from the file system.
func ReadTestFile(fs afero.Fs, filename string) (string, error) {
	data, err := afero.ReadFile(fs, filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CreateTestDirectory is a helper function to create test directories.
func CreateTestDirectory(fs afero.Fs, dirname string) error {
	return fs.MkdirAll(dirname, 0755)
}

// AssertNoError asserts that err is nil.
func AssertNoError(t *testing.T, err error, format string, args ...interface{}) {
	t.Helper()
	if err != nil {
		t.Errorf(format+" returns unexpected err: %+v", append(args, err)...)
	}
}

// AssertError asserts that err is not nil.
func AssertError(t *testing.T, err error, format string, args ...interface{}) {
	t.Helper()
	if err == nil {
		t.Errorf(format+" expects to return an error, but no error", args...)
	}
}

// AssertEqual asserts that got equals want using == comparison.
func AssertEqual(t *testing.T, got, want interface{}, format string, args ...interface{}) {
	t.Helper()
	if got != want {
		t.Errorf(format+" returns %v, but want = %v", append(args, got, want)...)
	}
}

// AssertDeepEqual asserts that got equals want using reflect.DeepEqual.
func AssertDeepEqual(t *testing.T, got, want interface{}, format string, args ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf(format+" returns %#v, but want = %#v", append(args, got, want)...)
	}
}

// AssertDiff asserts that got equals want using go-cmp and shows diff on failure.
func AssertDiff(t *testing.T, got, want interface{}, format string, args ...interface{}) {
	t.Helper()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(format+" mismatch:\ngot: %s\nwant: %s\ndiff: %s",
			append(args, spew.Sdump(got), spew.Sdump(want), diff)...)
	}
}

// AssertFileContent asserts that file content matches expected string.
func AssertFileContent(t *testing.T, fs afero.Fs, filename, expected, format string, args ...interface{}) {
	t.Helper()
	got, err := afero.ReadFile(fs, filename)
	if err != nil {
		t.Fatalf("failed to read file %s: %s", filename, err)
	}
	if string(got) != expected {
		t.Errorf(format+" file content mismatch:\ngot: %s\nwant: %s",
			append(args, string(got), expected)...)
	}
}

// AssertCommandSuccess asserts that command exit code is 0.
func AssertCommandSuccess(t *testing.T, exitCode int, format string, args ...interface{}) {
	t.Helper()
	if exitCode != 0 {
		t.Errorf(format+" returns exit code %d, but want = 0", append(args, exitCode)...)
	}
}

// AssertCommandFailure asserts that command exit code is not 0.
func AssertCommandFailure(t *testing.T, exitCode int, format string, args ...interface{}) {
	t.Helper()
	if exitCode == 0 {
		t.Errorf(format+" returns exit code 0, but want != 0", args...)
	}
}

// AssertUIOutput asserts that MockUI output contains expected content.
func AssertUIOutput(t *testing.T, ui *MockUI, expected, format string, args ...interface{}) {
	t.Helper()
	got := ui.GetOutput()
	if !strings.Contains(got, expected) {
		t.Errorf(format+" UI output does not contain expected content:\ngot: %s\nwant to contain: %s",
			append(args, got, expected)...)
	}
}

// AssertUIError asserts that MockUI error output contains expected content.
func AssertUIError(t *testing.T, ui *MockUI, expected, format string, args ...interface{}) {
	t.Helper()
	got := ui.GetErrorOutput()
	if !strings.Contains(got, expected) {
		t.Errorf(format+" UI error output does not contain expected content:\ngot: %s\nwant to contain: %s",
			append(args, got, expected)...)
	}
}
