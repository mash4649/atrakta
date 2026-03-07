package runtimeobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/metrics"
	"sort"
	"time"

	"atrakta/internal/util"
)

const (
	statsVersion = 1
	maxSamples   = 64
)

type statsFile struct {
	V            int                          `json:"v"`
	UpdatedAt    string                       `json:"updated_at"`
	SamplesMs    map[string][]int64           `json:"samples_ms"`
	SchedulerObs map[string][]SchedulerSample `json:"scheduler_obs,omitempty"`
}

type SchedulerSample struct {
	Goroutines         uint64  `json:"goroutines,omitempty"`
	GOMAXPROCS         uint64  `json:"gomaxprocs,omitempty"`
	SchedLatencyP50Use float64 `json:"sched_latency_p50_usec,omitempty"`
	SchedLatencyP95Use float64 `json:"sched_latency_p95_usec,omitempty"`
}

type Snapshot struct {
	Command   string
	LastMs    int64
	P50Ms     int64
	P95Ms     int64
	Count     int
	Scheduler SchedulerSample
}

func Record(repoRoot, command string, duration time.Duration) (Snapshot, error) {
	if command == "" {
		return Snapshot{}, fmt.Errorf("command required")
	}
	lock := lockPath(repoRoot)
	statsPath := path(repoRoot)
	var out Snapshot
	err := util.WithFileLock(lock, util.DefaultFileLockOptions(), func() error {
		sf, err := load(statsPath)
		if err != nil {
			return err
		}
		if sf.SamplesMs == nil {
			sf.SamplesMs = map[string][]int64{}
		}
		if sf.SchedulerObs == nil {
			sf.SchedulerObs = map[string][]SchedulerSample{}
		}
		ms := duration.Milliseconds()
		if ms < 0 {
			ms = 0
		}
		h := append(sf.SamplesMs[command], ms)
		if len(h) > maxSamples {
			h = h[len(h)-maxSamples:]
		}
		sf.SamplesMs[command] = h
		sched := collectSchedulerMetrics()
		sh := append(sf.SchedulerObs[command], sched)
		if len(sh) > maxSamples {
			sh = sh[len(sh)-maxSamples:]
		}
		sf.SchedulerObs[command] = sh
		sf.V = statsVersion
		sf.UpdatedAt = util.NowUTC()
		if err := save(statsPath, sf); err != nil {
			return err
		}
		p50, p95 := percentiles(h)
		out = Snapshot{
			Command:   command,
			LastMs:    ms,
			P50Ms:     p50,
			P95Ms:     p95,
			Count:     len(h),
			Scheduler: sched,
		}
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	return out, nil
}

func path(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "metrics", "runtime.json")
}

func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", ".locks", "runtime.metrics.lock")
}

func load(path string) (statsFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return statsFile{V: statsVersion, SamplesMs: map[string][]int64{}}, nil
		}
		return statsFile{}, fmt.Errorf("read runtime metrics: %w", err)
	}
	var sf statsFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return statsFile{}, fmt.Errorf("parse runtime metrics: %w", err)
	}
	if sf.V != statsVersion {
		return statsFile{}, fmt.Errorf("runtime metrics v must be %d", statsVersion)
	}
	if sf.SchedulerObs == nil {
		sf.SchedulerObs = map[string][]SchedulerSample{}
	}
	return sf, nil
}

func save(path string, sf statsFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir metrics dir: %w", err)
	}
	b, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime metrics: %w", err)
	}
	b = append(b, '\n')
	if err := util.AtomicWriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write runtime metrics: %w", err)
	}
	return nil
}

func percentiles(values []int64) (p50 int64, p95 int64) {
	if len(values) == 0 {
		return 0, 0
	}
	c := append([]int64(nil), values...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	p50 = c[(len(c)-1)*50/100]
	p95 = c[(len(c)-1)*95/100]
	return p50, p95
}

func collectSchedulerMetrics() SchedulerSample {
	samples := []metrics.Sample{
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/sched/gomaxprocs:threads"},
		{Name: "/sched/latencies:seconds"},
	}
	metrics.Read(samples)
	out := SchedulerSample{}
	if samples[0].Value.Kind() == metrics.KindUint64 {
		out.Goroutines = samples[0].Value.Uint64()
	}
	if samples[1].Value.Kind() == metrics.KindUint64 {
		out.GOMAXPROCS = samples[1].Value.Uint64()
	}
	if samples[2].Value.Kind() == metrics.KindFloat64Histogram {
		h := samples[2].Value.Float64Histogram()
		out.SchedLatencyP50Use = histogramQuantileUS(h, 0.50)
		out.SchedLatencyP95Use = histogramQuantileUS(h, 0.95)
	}
	return out
}

func histogramQuantileUS(h *metrics.Float64Histogram, q float64) float64 {
	if h == nil || len(h.Counts) == 0 || len(h.Buckets) == 0 {
		return 0
	}
	var total uint64
	for _, c := range h.Counts {
		total += c
	}
	if total == 0 {
		return 0
	}
	target := uint64(float64(total)*q + 0.5)
	if target < 1 {
		target = 1
	}
	var acc uint64
	for i, c := range h.Counts {
		acc += c
		if acc >= target {
			if i+1 < len(h.Buckets) {
				return h.Buckets[i+1] * 1e6
			}
			return h.Buckets[i] * 1e6
		}
	}
	last := h.Buckets[len(h.Buckets)-1]
	return last * 1e6
}
