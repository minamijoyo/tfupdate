# Contributing to tfupdate

Thank you for your interest in contributing to tfupdate! This document provides comprehensive guidelines for all contributors, including human developers and AI agents.

> **Note for AI Agents**: This document contains the authoritative development standards. Also refer to `CLAUDE.md` for AI-specific workflow guidance.

## Development Setup

### Prerequisites

- Go 1.24+
- golangci-lint (for linting)
- Make (for build automation)

### Build and Test

- `make build` - Build the binary
- `make test` - Run unit tests
- `make testacc` - Run acceptance tests (requires install first)
- `make lint` - Run linting
- `make check` - Run both lint and test

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
- Use struct with `desc`, `filename`, `src`, `version`, `want`, `ok` fields
- Use `t.Run(tc.desc, func(t *testing.T) {...})` for sub-tests
- Include both positive and negative test cases

### Mock Naming and Patterns

**Mock Struct Naming:**
- Pattern: `mock<InterfaceName>` or `mock<TypeName>Client`
- Examples: `mockGitHubClient`, `mockTFRegistryClient`

**Interface Compliance:**
- Verify mock implements interface with `var _ InterfaceName = (*mockStructName)(nil)`

### Sub-test Naming

- Use descriptive scenario names: `"simple"`, `"error case"`, `"invalid input"`
- Describe test scenario, not implementation details
- Examples: `"simple"`, `"sort"`, `"pre-release"`, `"no release"`, `"api error"`

### Error Testing

- Check `tc.ok && err != nil` for unexpected errors
- Check `!tc.ok && err == nil` for missing expected errors
- Use descriptive error messages with input context

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
