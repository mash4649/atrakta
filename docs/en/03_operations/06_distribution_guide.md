# Distribution Guide

[English](./06_distribution_guide.md) | [日本語](../../ja/03_運用/06_配布手順.md)

## Policy

- Distribute prebuilt binaries to end users (Go not required)
- Keep source builds as optional path for developers/maintainers
- Do not commit distributable binaries into repository
- Auto-generate releases on push to `main` (`.github/workflows/release.yml`)
- Use CalVer for release tags (`YYYY.MM.DD-N`) and keep product version in `VERSION` (SemVer)

## Automated Release

- Triggers:
  - `push` to `main`
  - `workflow_dispatch`
- Steps:
  1. Determine next CalVer tag from latest release
  2. Build artifacts with `scripts/build_release_artifacts.sh`
  3. Generate release notes from `.github/release.yml`
  4. Create GitHub Release with packages and checksums

## Build Artifacts

```bash
./scripts/build_release_artifacts.sh
```

Outputs:

- `.tmp/release/v<version>/packages`
- `.tmp/release/v<version>/checksums.txt`

## Publication Steps

1. Normally just verify the auto-release result
2. On failure, rerun via `workflow_dispatch` or build manually
3. Announce user update procedure (binary replacement -> `atrakta doctor`)

## User Onboarding Template

1. macOS / Linux users can run one-command installer:
   - `curl -fsSL https://raw.githubusercontent.com/afwm/Atrakta/main/scripts/install.sh | bash`
2. Download OS/arch archive from Releases (manual path)
   - Extracted executable name is `atrakta` (`atrakta.exe` on Windows)
3. Put `atrakta` (`atrakta.exe` on Windows) on PATH
   - macOS / Linux:
     - `mkdir -p ~/.local/bin`
     - `install -m 0755 ./atrakta ~/.local/bin/atrakta`
     - `hash -r`
   - Windows (PowerShell):
     - `$targetDir = "$env:USERPROFILE\AppData\Local\Programs\atrakta"`
     - `New-Item -ItemType Directory -Force $targetDir | Out-Null`
     - `Copy-Item .\atrakta.exe "$targetDir\atrakta.exe" -Force`
     - add `$targetDir` to user PATH
4. Verify with `atrakta --help`
5. Run once in target project: `atrakta init --interfaces <primary>`
