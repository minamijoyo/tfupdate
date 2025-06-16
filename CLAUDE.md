# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tfupdate is a CLI tool that updates version constraints for Terraform and OpenTofu configurations. It can update core versions, providers, modules, and dependency lock files without requiring Terraform/OpenTofu CLI.

## Quick Reference

See @CONTRIBUTING.md for comprehensive development setup, testing guidelines, and build instructions.

### Essential Commands
```bash
make check          # Run lint + test (use before each commit)
make test           # Unit tests only
make testacc        # Acceptance tests
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

## AI Agent Development Rules

When working on this codebase, AI agents must follow these specific guidelines:

### Core Requirements
- Refer to @CONTRIBUTING.md for development standards and testing conventions
- Code, comments, and documentation must be written in English
- Follow established patterns and maintain consistency with existing codebase

### AI-Specific Workflow Rules
- Use phase documents (.claude/projects/{project}/phase*.md) for complex tasks
- Make incremental commits with descriptive messages
- Update phase document checklists after task completion
- Prioritize code readability and comprehensive documentation

### Project Organization
- Store AI-specific planning documents in `.claude/projects/{project_name}/`
- Use Architecture Decision Records (ADR) in `docs/adr/` with YYYYMMDD prefix for technical decisions
- Maintain clear separation between general project docs and AI optimization work

### Quality Verification
- Run `make check` after implementation before committing
- Fix lint and test issues before committing
- Ensure test coverage meets project standards (>85% for new code)
- Verify all existing tests pass without regression

## Active Projects

This section tracks ongoing improvement projects for the codebase.

### AI Agent Optimization
Optimizing the codebase for AI agent coding efficiency through improved test coverage, documentation, and development experience.

**Resources:**
- Task list: `.claude/projects/ai_agent_optimization/TODO.md`
- Analysis & plan: `docs/adr/20250614_ai_agent_optimization.md`
- Phase 1 plan: `.claude/projects/ai_agent_optimization/phase1.md`

**Scope:** Test coverage improvements, architecture documentation, code readability enhancements, and development tooling standardization.
