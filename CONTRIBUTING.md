# Contributing Guide

Thanks for your interest in contributing to Trust-Tunnel! This document will guide you through the process.

## Getting Started

### Prerequisites

- Go 1.21+
- Docker (for building images and running E2E tests)
- Linux environment (required for E2E tests and container-related features)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/antgroup/trust-tunnel.git
cd trust-tunnel

# Build binaries
make trust-tunnel-agent && make trust-tunnel-client

# Build Docker images
make images

# Run unit tests
go test ./...

# Run static analysis
go vet ./...

# Run E2E tests (requires Linux + Docker)
cd e2e && go test -v .
```

## How to Contribute

### Reporting Bugs

1. Check existing issues to avoid duplicates
2. Open a new issue using the [Bug Report template](.github/ISSUE_TEMPLATE/bug_report.md)
3. Include environment details, reproduction steps, and expected vs. actual behavior

### Requesting Features

1. Open a new issue using the [Feature Request template](.github/ISSUE_TEMPLATE/feature_request.md)
2. Describe the problem you're trying to solve, not just the solution you want

### Submitting Changes

1. **Fork** the repository
2. **Create a branch** from `main`: `git checkout -b feature/your-feature`
3. **Make your changes** following our coding standards
4. **Test your changes**: `go build ./... && go vet ./... && go test ./...`
5. **Commit** with a clear message following [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat: add new authentication plugin`
   - `fix: resolve container session error message formatting`
   - `refactor: extract DockerClient interface`
   - `docs: update README`
6. **Push** to your fork: `git push origin feature/your-feature`
7. **Open a Pull Request** against the `main` branch

## Coding Standards

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Run `gofmt -w .` before committing
- Run `go vet ./...` and fix all warnings
- No panics in library code — return errors instead
- Add tests for new functionality

### Code Review Policy

Maintainers review PRs with the following priorities:

1. Check the related issue and PR description
2. Verify solution readability and simplicity
3. Check test coverage for the change
4. Pay attention to:
   - Code structure changes
   - Error handling
   - Edge cases and concurrency issues
   - Breaking changes (avoid unless justified)

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `ci`

### Pull Request Checklist

- [ ] Code follows the style guidelines
- [ ] Self-review completed
- [ ] Comments added for hard-to-understand code
- [ ] Documentation updated if needed
- [ ] No new warnings generated
- [ ] Tests added for bug fixes or new features
- [ ] All tests pass locally

## Architecture Overview

Trust-Tunnel has three main components:

- **Agent** (`cmd/trust-tunnel-agent/`): Runs on each node, accepts WebSocket connections
- **Client** (`cmd/trust-tunnel-client/`): CLI tool for connecting to Agent
- **Auth Server** (`pkg/trust-tunnel-agent/auth/`): Optional authentication plugin system

Key design patterns:

- **Interface-based decoupling**: External dependencies (Docker SDK, containerd) are wrapped behind interfaces (`DockerClient`, `APIClient`) for testability
- **Session abstraction**: All connection modes (Docker sidecar, Docker exec, containerd, nsenter, SSH) implement the `Session` interface
- **Plugin authentication**: Auth handlers are registered via factory pattern

For detailed architecture, see `.claude/rules/architecture.md`.

## Releasing

Maintainers follow this process for releases:

1. Update `CHANGELOG.md` with the new version
2. Tag the release: `git tag v0.x.0`
3. Push the tag: `git push origin v0.x.0`
4. Create a GitHub Release with release notes from CHANGELOG

## License

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).