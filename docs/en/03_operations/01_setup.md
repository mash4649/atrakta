# Setup Guide

[English](./01_setup.md) | [日本語](../../ja/03_運用/01_導入手順.md)

## 1. Install Binary

Place `atrakta` on PATH.

Binary distribution is the default. Recommended locations:

- macOS / Linux: `~/.local/bin/atrakta`
- Windows: `%USERPROFILE%\\AppData\\Local\\Programs\\atrakta\\atrakta.exe`

Recommended one-command install (macOS / Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/afwm/Atrakta/main/scripts/install.sh | bash
```

`install.sh` auto-detects OS/arch, downloads the matching release asset, verifies checksum, installs to `~/.local/bin`, and removes macOS quarantine attribute.

Installation examples after downloading from Releases:
The extracted executable name is unified as `atrakta` (`atrakta.exe` on Windows).

```bash
# macOS / Linux
mkdir -p ~/.local/bin
install -m 0755 ./atrakta ~/.local/bin/atrakta
hash -r
atrakta --help
```

```powershell
# Windows (PowerShell)
$targetDir = "$env:USERPROFILE\AppData\Local\Programs\atrakta"
New-Item -ItemType Directory -Force $targetDir | Out-Null
Copy-Item .\atrakta.exe "$targetDir\atrakta.exe" -Force
$userPath = [Environment]::GetEnvironmentVariable("Path","User")
if ($userPath -notlike "*$targetDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$targetDir;$userPath", "User")
}
where atrakta
```

Optional source build in the target project:

```bash
cd <project>/atrakta
go build -o ~/.local/bin/atrakta ./cmd/atrakta
hash -r
atrakta --help
```

If `command not found`:

```bash
echo "$PATH" | tr ':' '\n' | grep "$HOME/.local/bin"
ls -l ~/.local/bin/atrakta
```

Go is not required when using prebuilt binaries.

## 2. First-time Setup

Run at project root:

```bash
# uniform command for both interactive and non-interactive usage
atrakta init --interfaces cursor
```

Generated/updated on first run as needed:

- `AGENTS.md`
- `.atrakta/contract.json`
- `.atrakta/state.json`
- `.atrakta/events.jsonl`
- `.atrakta/progress.json`
- `.atrakta/task-graph.json` (when plan runs)
- `.atrakta/policies/prompt-min.json` (when default policy is referenced)

## 3. Recommended Optional Setup

- After the single command above, `start` is usually auto-resolved via wrapper/hook/IDE autostart tasks.
- If `--interfaces` is omitted, resolution falls back to prompt or `needs_input` (no implicit default).

For staged manual setup:

```bash
atrakta wrap install
atrakta hook install
atrakta ide-autostart install
atrakta start --interfaces cursor
```
