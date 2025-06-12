# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tfupdate is a CLI tool that updates version constraints for Terraform and OpenTofu configurations. It can update core versions, providers, modules, and dependency lock files without requiring Terraform/OpenTofu CLI.

## Development Commands

### Build and Install
```bash
make build          # Build binary to bin/tfupdate
make install        # Install to $GOPATH/bin
go install          # Alternative install method
```

### Testing
```bash
make test           # Run unit tests
make testacc        # Run acceptance tests (requires install first)
make testacc-all    # Run all acceptance test suites
make lint           # Run golangci-lint
make check          # Run both lint and test
```

### Dependencies
```bash
make deps           # Download Go modules
go mod download     # Alternative dependency download
```

## Code Architecture

### Core Components

**Command Layer** (`/command/`)
- CLI interface using Mitchell Hashimoto's CLI library
- Commands: `terraform.go`, `opentofu.go`, `provider.go`, `module.go`, `lock.go`, `release.go`

**Business Logic** (`/tfupdate/`)
- Main updater interfaces and implementations
- `update.go` - Factory pattern for creating appropriate updaters
- `context.go`, `option.go` - Configuration management
- `file.go`, `hclwrite.go` - HCL file manipulation using `github.com/hashicorp/hcl/v2`

**Version Management** (`/release/`)
- Integrations with GitHub, GitLab, Terraform Registry, OpenTofu Registry
- `version.go` - Version parsing with `github.com/hashicorp/go-version`

**Registry Client** (`/tfregistry/`)
- HTTP clients for Terraform and OpenTofu registries
- Separate provider and module API handlers

**Lock File Handling** (`/lock/`)
- Dependency lock file updates without CLI tools
- Provider package downloading and hash calculation
- Performance-optimized with in-memory caching

### Key Patterns

- **Interface-driven design** - Heavy use of interfaces for testability (Updater, Release, etc.)
- **Factory pattern** - `tfupdate.NewUpdater()` creates appropriate updater based on type
- **Mock implementations** - Comprehensive mocking for external dependencies
- **Context-aware operations** - Proper error handling and context propagation

### File Types Handled

- **HCL files** (`.tf`) - Terraform/OpenTofu configurations
- **Lock files** (`.terraform.lock.hcl`) - Dependency constraints and hashes
- **Registry sources** - GitHub releases, GitLab releases, Terraform/OpenTofu registries

## Environment Variables

```bash
GITHUB_TOKEN=<token>              # For private GitHub repositories
GITLAB_TOKEN=<token>              # For GitLab repositories (with api permissions)
GITLAB_BASE_URL=<url>             # For GitLab instances (default: https://gitlab.com/api/v4/)
TFREGISTRY_BASE_URL=<url>         # For OpenTofu registry (set to https://registry.opentofu.org/)
```

## Test Infrastructure

- **Unit tests** - Comprehensive coverage with `*_test.go` files
- **Integration tests** - Acceptance tests in `/scripts/testacc/`
- **Test fixtures** - Sample configurations in `/test-fixtures/`
- **Mock interfaces** - Isolated testing of external dependencies
- **CI matrix testing** - Multiple Terraform (0.14.11-1.11.3) and OpenTofu (1.6.3-1.9.1) versions

## Dependencies

Key external libraries:
- `github.com/hashicorp/hcl/v2` - HCL parsing and writing
- `github.com/mitchellh/cli` - Command-line interface framework
- `github.com/hashicorp/go-version` - Semantic versioning
- `github.com/google/go-github/v28` - GitHub API client
- `github.com/xanzy/go-gitlab` - GitLab API client
- `github.com/spf13/afero` - Abstract file system interface (for testing)

## Build Requirements

- Go 1.24+
- golangci-lint (for linting)
- Make (for build automation)