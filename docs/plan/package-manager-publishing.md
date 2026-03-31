# Package Manager Publishing

Atrakta is shipped as a Go binary. Package manager publishing is a distribution
wrapper around that binary, not a second source of truth.

## Goal

Make the release artifacts easy to consume from the ecosystems people already
use:

- npm
- pip
- cargo

The package manager surface should:

- install a versioned Atrakta binary
- keep the CLI behavior identical to the released binary
- defer to GitHub Releases assets for the actual payload
- avoid introducing new runtime dependencies into the core Go module

## Publishing Model

Use the package manager only as a transport and launch layer.

- The canonical release artifact remains the platform binary from `goreleaser`
  and GitHub Releases.
- Package metadata should be minimal and versioned alongside the release.
- Updates should follow the same semantic version as the tagged binary.
- Checksums and provenance should continue to be validated at download time.

## Ecosystem Notes

### npm

Prefer a thin wrapper package whose install step resolves the matching binary
for the current platform.

### pip

Prefer a command-line package that installs the binary as a tool entrypoint.

### cargo

Prefer a wrapper crate that launches the released binary instead of rebuilding
the Go code.

## Safety Rules

- Do not publish source code as the package payload.
- Do not bypass the release binary for normal installs.
- Do not add ecosystem-specific runtime libraries to the Go module.
- Keep package metadata small, deterministic, and easy to audit.

## Current Status

Package manager publishing is a Phase 4 distribution task. The repository
documents the target shape now, but implementation should wait until release
artifacts and installer flows are stable.
