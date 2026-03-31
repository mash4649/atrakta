# Error Catalog

This catalog defines the structured `error` envelope used by `atrakta` JSON outputs.
When a command exits with code `1` and `--json` is enabled, the top-level payload includes:

```json
{
  "status": "error",
  "error": {
    "code": "ERR_*",
    "message": "human-readable failure",
    "recovery_steps": ["..."]
  }
}
```

| Code | Meaning | Recovery |
|------|---------|----------|
| `ERR_USAGE` | Invalid CLI usage, unsupported subcommand, or missing required flag/argument. | Re-run the command with `--help` and provide the required flag(s) or positional argument(s). |
| `ERR_APPROVAL_REQUIRED` | The command reached a write path that requires explicit approval. | Re-run with `--approve` or use an interactive terminal to confirm the action, then review the proposed write set. |
| `ERR_BLOCKED` | The command is blocked by repository state, a handoff, or a deny condition. | Inspect the reported state or handoff, resolve the blocker, then retry the command. |
| `ERR_NOT_FOUND` | A required file or managed artifact was missing. | Restore the file or rerun the bootstrap command that creates it. |
| `ERR_RUNTIME` | An unexpected runtime failure occurred. | Inspect logs/output, fix the underlying issue, then retry. |
