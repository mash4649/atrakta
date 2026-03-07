# Verification Commands

[English](./01_verification_commands.md) | [日本語](../../ja/04_品質/01_検証コマンド.md)

## Basic Verification

```bash
./scripts/dev/verify_loop.sh
```

A minimal verification loop that retries once with `doctor` when `go test ./...` fails.

## Provisional Verification

```bash
./scripts/dev/verify_provisional.sh
```

Includes:

- `gofmt` + full tests
- wrapper fast-path benchmark threshold check (`<50ms`)
- Linux/Windows cross `go test -run '^$'`
- Linux/Windows binary build confirmation

## Phase2 Verification

```bash
./scripts/dev/verify_phase2.sh
```

Additional checks:

- detect/apply scaling benchmarks
- p95/p99 performance regression gate (fail-closed)
- projection linear scaling verification
- H7 migration replay determinism test
- events schema v2 integrity test (`migrate check`)
- Docker runtime verification for Linux binary
- Windows runtime test is skipped under environment constraints

## Performance Regression Gate

```bash
./scripts/dev/verify_perf_gate.sh
```

Covers:

- p95/p99 thresholds for `BenchmarkApplyScaling/ops_300`
- p95/p99 thresholds for `BenchmarkBuildNoopManagedScaling/managed_1000`
- p95/p99 thresholds for `BenchmarkWrapperFastPath`
- p95/p99 thresholds for `BenchmarkStartSteadyState`
- `TestSLORepoMapTokenBudgetRespected` (token budget)
- `TestSLOWrapperFastPathHitRate` (fast-path hit rate)
- immediate failure on threshold breach (fail-closed)

CI usage:

- GitHub Actions (`.github/workflows/ci.yml`) runs `verify_loop` + `verify_perf_gate` on every `push/pull_request`

## Fault Injection Tests

- `go test ./internal/events -run '^TestFaultInjection'`
- `go test ./internal/util -run '^TestFaultInjectionWithFileLock'`
- Included in `verify_loop`, therefore always executed in normal CI

## Soak (Long-running)

```bash
# short measurement with custom minutes
./scripts/dev/soak.sh --minutes 10

# 24h / 72h profiles
./scripts/dev/soak_24h.sh
./scripts/dev/soak_72h.sh
```

Notes:

- 24h/72h does not complete on GitHub Hosted Runners due to job time limits
- run on self-hosted environments for full execution
