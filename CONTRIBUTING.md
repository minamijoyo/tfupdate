# Contributing to tfupdate

Thank you for your interest in contributing to tfupdate! This document provides comprehensive guidelines for all contributors, including human developers and AI agents.

> **Note for AI Agents**: This document contains the authoritative development standards. Also refer to `CLAUDE.md` for AI-specific workflow guidance.

## Development Setup

### Prerequisites

- Go 1.24+
- golangci-lint (for linting)
- Make (for build automation)

### Build and Test

```bash
# Build the binary
make build

# Run unit tests
make test

# Run acceptance tests (requires install first)
make testacc

# Run linting
make lint

# Run both lint and test
make check
```

## Code Style and Standards

### Language Requirements

- **ALL code, comments, documentation, and commit messages MUST be written in English**
- Use clear, descriptive names for variables, functions, and types
- Follow standard Go conventions (gofmt, golint, etc.)

### Architecture Patterns

- **Interface-driven design**: Use interfaces for testability and modularity
- **Factory pattern**: Use factory functions for creating instances with dependencies
- **Context-aware operations**: Proper error handling and context propagation
- **Mock implementations**: Comprehensive mocking for external dependencies

### Code Quality

- Maintain high test coverage (aim for >85% on new code)
- Use proper error handling with context propagation
- Add comprehensive unit tests for all new functionality
- Follow existing code patterns and architectural decisions

## Testing Guidelines

### Test File Naming Conventions

**Test File Naming:**
- Follow pattern: `<source_file>_test.go` (e.g., `terraform.go` → `terraform_test.go`)
- Place test files in the same package as source files

**Test Function Naming:**
- Constructor tests: `TestNew<TypeName>` (e.g., `TestNewTerraformCommand`)
- Method tests: `Test<TypeName><MethodName>` (e.g., `TestTerraformCommandRun`)
- Function tests: `Test<FunctionName>` (e.g., `TestNewRelease`)

### Test Case Patterns

**Table-Driven Tests:**
```go
cases := []struct {
    desc     string // Description for sub-tests
    filename string // Input parameters
    src      string
    version  string
    want     string // Expected results
    ok       bool   // Expected success/failure
}{
    {
        desc:     "simple case",
        filename: "test.tf",
        src:      "terraform { required_version = \"0.12.0\" }",
        version:  "0.13.0",
        want:     "terraform { required_version = \"0.13.0\" }",
        ok:       true,
    },
    // ... more test cases
}

for _, tc := range cases {
    t.Run(tc.desc, func(t *testing.T) {
        // Test implementation using tc fields
    })
}
```

**Test Case Structure:**
- Use descriptive field names: `filename`, `src`, `version`, `want`, `ok`
- Use `desc` field for sub-test names with `t.Run()`
- Include both positive (`ok: true`) and negative (`ok: false`) test cases

### Mock Naming and Patterns

**Mock Struct Naming:**
- Pattern: `mock<InterfaceName>` or `mock<TypeName>Client`
- Examples: `mockGitHubClient`, `mockTFRegistryClient`

**Interface Compliance:**
```go
// Verify mock implements the interface
var _ InterfaceName = (*mockStructName)(nil)
```

### Sub-test Naming

- Use descriptive scenario names: `"simple"`, `"error case"`, `"invalid input"`
- Describe test scenario, not implementation details
- Examples: `"simple"`, `"sort"`, `"pre-release"`, `"no release"`, `"api error"`

### Error Testing

```go
// Standard error checking pattern
if tc.ok && err != nil {
    t.Errorf("FunctionName() with input = %s returns unexpected err: %+v", input, err)
}
if !tc.ok && err == nil {
    t.Errorf("FunctionName() with input = %s expects to return an error, but no error", input)
}
```

### Test Infrastructure

**File System Testing:**
- Use `afero.NewMemMapFs()` for in-memory file system testing
- Avoid external file dependencies for unit tests
- Use test fixtures in `/test-fixtures/` only for acceptance tests

**Assertions:**
- Use `command.Assert*` helper functions for consistent error messages
- Use `reflect.DeepEqual()` for simple comparisons
- Use `github.com/google/go-cmp/cmp` for complex struct comparisons
- Use `github.com/davecgh/go-spew/spew` for detailed debugging output

**Mock External Dependencies:**
- Mock HTTP clients for GitHub, GitLab, Terraform Registry APIs
- Use in-memory file systems for file operations
- Isolate external dependencies consistently

## Commit Guidelines

### Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New features or functionality
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring without functional changes
- `test`: Adding or updating tests
- `perf`: Performance improvements
- `ci`: Changes to CI configuration files and scripts
- `build`: Changes to build system or external dependencies
- `chore`: Other changes that don't modify src or test files

**Examples:**
```
feat: add support for OpenTofu registry provider updates

Add new provider update functionality that supports both Terraform
and OpenTofu registries with configurable base URLs.

fix: handle edge case in version constraint parsing

Fix issue where pre-release versions were not properly handled
in version constraint updates.

test: add unit tests for TerraformCommand help methods

docs: update commit message format to conventional commits
```

### Commit Requirements

- Write clear, descriptive commit messages
- One logical change per commit
- Include tests for new functionality
- Ensure all tests pass before committing
- Run `make check` to verify code quality

## Pull Request Process

1. **Fork and Branch**: Create a feature branch from `master`
2. **Implement**: Follow coding standards and write tests
3. **Test**: Ensure all tests pass and coverage is maintained
4. **Lint**: Run `make lint` and fix any issues
5. **Commit**: Use clear commit messages following the guidelines
6. **Pull Request**: Create a PR with descriptive title and summary

### Pull Request Requirements

- Include comprehensive tests for new functionality
- Maintain or improve test coverage
- Update documentation as needed
- Pass all CI checks
- Include clear description of changes and rationale

## Development Workflow

### Test-Driven Development (TDD)

1. Write tests before implementing features
2. Ensure tests fail initially (red)
3. Implement minimal code to make tests pass (green)
4. Refactor while keeping tests passing (refactor)

### Code Review

- All changes require code review
- Address feedback constructively
- Ensure CI passes before merging
- Maintain backward compatibility when possible

## Architecture Guidelines

### Package Structure

- **`/command/`**: CLI interface and command implementations
- **`/tfupdate/`**: Core business logic and updater interfaces
- **`/release/`**: Version management and registry integrations
- **`/tfregistry/`**: HTTP clients for Terraform/OpenTofu registries
- **`/lock/`**: Dependency lock file handling

### Interface Design

- Design interfaces for testability and modularity
- Keep interfaces focused and cohesive
- Use dependency injection for external dependencies
- Provide mock implementations for testing

### Error Handling

- Use context for cancellation and timeouts
- Provide meaningful error messages
- Wrap errors with additional context when appropriate
- Handle edge cases gracefully

## Getting Help

- Create an issue for bugs or feature requests
- Join discussions in existing issues
- Follow the project's code of conduct
- Refer to existing code for patterns and examples

Thank you for contributing to tfupdate!