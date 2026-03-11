# Benchmark Criteria

[English](./02_benchmark_criteria.md) | [日本語](../../ja/04_品質/02_ベンチマーク基準.md)

## Fast Path

- Metric: `BenchmarkWrapperFastPath`
- Pass condition: `ns/op < 50,000,000` (under 50ms)
- Method: run with `-count=5` and evaluate extracted final value

## SLO Gate (fail-closed)

- `BenchmarkApplyScaling/ops_300`: p95/p99
- `BenchmarkBuildNoopManagedScaling/managed_1000`: p95/p99
- `BenchmarkWrapperFastPath`: p95/p99
- `BenchmarkStartSteadyState`: p95/p99
- `TestSLORepoMapTokenBudgetRespected`: `repo_map.used_tokens <= budget`
- `TestSLOWrapperFastPathHitRate`: `hit_rate >= 0.95`

Notes:

- `BenchmarkStartSteadyState` measures steady state with snapshot fast gate enabled
- occasional slower spikes are expected when strict path is triggered by strict interval

## Extended Benchmarks (Phase2)

- `BenchmarkDetectScaling`
- `BenchmarkApplyScaling`
- `BenchmarkBuildNoopManagedScaling`
- `BenchmarkStartSteadyState`
- `BenchmarkWrapperFastPath`
- `BenchmarkProjectionRender`
- `BenchmarkParityDoctor`
- `BenchmarkProjectionRepair`
- `BenchmarkExtensionRender`
- `TestProjectionScalingLinearBound`
- `TestParityConsistencyRate`
- `TestParityDriftFalsePositiveRate`
- `TestParityDriftFalseNegativeRate`
- `TestBrownfieldAppendIdempotent`
- `TestBrownfieldNoOverwrite`

## Operational Rules

- fail immediately when thresholds are violated
- evaluate p95/p99 + token/hit-rate in `verify_perf_gate.sh` (fail-closed)
- maintain determinism and safety gates during speed optimization
- keep fast-path for no-dependency cases even with DAG task graph
- verify apply scaling remains acceptable after edit safety (Go AST / JSON parse)
- keep `repair` out of the default `start` critical path (run via explicit commands)
