# Changelog

All notable changes to this project are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Git collaboration docs under `docs/git/` (contributing, branching, commits, releases)
- GitHub PR template, issue templates, and `CODEOWNERS`

### Changed

### Fixed

### Security

## [0.1.0] - 2026-08-10

### Added

- Modular monolith layout: `domain`, `service`, `repository`, `handlers`, `server`
- Shared `apperr` error types and JSON `response` envelope
- CORS and request-ID middleware (generates ID when client omits it)
- Health and API welcome endpoints with version from config
- Config for `APP_VERSION`, `CORS_ALLOWED_ORIGINS`, `CORS_ALLOW_CREDENTIALS`
- Middleware and `apperr` unit tests

### Changed

- README aligned with real project structure (no longer claims missing middleware/hot-reload)

[Unreleased]: https://github.com/kidusshoa/asnakech-servers/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/kidusshoa/asnakech-servers/releases/tag/v0.1.0
