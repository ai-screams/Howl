# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Local CHANGELOG.md now manually synchronized with GitHub Releases

## [1.3.0] - 2026-02-10

### Added

- GitHub App authentication for auto-release workflow (replaces PAT)
- Complete release setup documentation (docs/RELEASE_SETUP.md)

### Changed

- Auto-release now uses GitHub App tokens (1-hour auto-expiry)
- Improved security: repository-scoped permissions, better audit trail

## [1.2.0] - 2026-02-10

### Added

- Modular reusable workflows for CI/CD pipeline (9 workflows)
- Security scanning: govulncheck + Gitleaks CLI + weekly audit
- Auto-release with semantic versioning (svu)
- Pre-commit hooks: go mod tidy + conventional commits validation
- Comprehensive godoc comments for all exported symbols

### Fixed

- CI/CD workflow failures: govulncheck, gitleaks, release-build
- Release build dependency: add test to prevent broken releases
- Upgrade Go to 1.24.13 for TLS/x509 security patches (GO-2025-3420, GO-2025-3373)

### Changed

- Coverage threshold: adjusted to 75% (actual 80.2%)
- Disable Go module cache (zero external dependencies)
- Tests disabled in pre-commit hook (CI-only per expert panel)

## [1.1.0] - 2026-02-08

### Added

- Support M suffix for 1M+ context window display
- Account email display for multi-account identification
- Claude Code plugin structure (Phase 1a)

### Fixed

- Migrate golangci-lint config to v2 schema
- Upgrade golangci-lint-action v6 to v7

### Changed

- Code cleanup and stdlib usage improvements

## [1.0.0] - 2026-02-07

### Added

- Core data types and stdin JSON parsing
- Derived metrics calculations (cost velocity, cache efficiency, API wait ratio)
- ANSI rendering with adaptive layout (2-4 lines normal, 2 lines danger mode)
- Git status display integration
- OAuth usage quota tracking (5h/7d)
- Transcript parsing for tools/agents display
- Release automation with GoReleaser and GitHub Actions
- Comprehensive unit tests (78% coverage)

### Changed

- Extract magic numbers to constants
- Remove deprecated and unused render functions

[Unreleased]: https://github.com/ai-screams/Howl/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/ai-screams/Howl/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/ai-screams/Howl/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/ai-screams/Howl/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/ai-screams/Howl/releases/tag/v1.0.0
