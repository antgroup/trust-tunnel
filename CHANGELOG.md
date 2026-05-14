# Changelog

All notable changes to this project will be documented in this file. We follow the Semantic Versioning 2.0.0 format.

## [0.2.0] - 2025-05-14

### Changed
- **Extract DockerClient interface**: Introduce `session.DockerClient` interface (14 methods) and `DockerClientAdapter` to decouple business logic from Docker SDK direct dependency, enabling unit test mocking
- **Introduce sidecar.APIClient interface**: Minimal 4-method interface in sidecar package to avoid circular import between session and sidecar
- **Add DockerClientFunc mock adapter**: Function-style mock with per-method Func fields, nil returns zero values
- **Improve code readability in core modules**: Extract magic numbers to named constants, improve variable naming, split Handler.Handle into smaller helper functions
- **Fix context leak in containerd.go**: Add `defer cancel()` for `context.WithCancel` in `execContainerd`

### Fixed
- Fix `checkSidecarNum` return value bug in handler.go
- Fix `err` variable shadowing in `NewHandler` (sidecar.Init error assignment)

## [0.1.0] - 2025-05-07

### Added
- Initial release of Trust-Tunnel secure access tool
- Trust-Tunnel Agent component for node-level secure connections
- Trust-Tunnel Client for end-user connections to remote resources
- Auth Server for managing user permissions
- Sandbox environment for secure command execution
- Support for secure connections to remote containers
- Support for secure connections to physical hosts
- Interactive terminal mode with `-it` flag
- Helm chart deployment for Trust-Tunnel Agent
- Custom permission system for access management
- Custom authorization plugin support
- End-to-end testing suite
