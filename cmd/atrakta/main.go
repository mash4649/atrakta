package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atrakta/internal/adapter"
	"atrakta/internal/bootstrap"
	"atrakta/internal/checkpoint"
	agentsctx "atrakta/internal/context"
	"atrakta/internal/contract"
	"atrakta/internal/core"
	"atrakta/internal/doctor"
	gcengine "atrakta/internal/gc"
	"atrakta/internal/hooks"
	"atrakta/internal/ide"
	"atrakta/internal/migrate"
	"atrakta/internal/model"
	"atrakta/internal/runtimeobs"
	"atrakta/internal/syncpolicy"
	"atrakta/internal/wrapper"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	ad := adapter.NewCLIAdapter()
	cwd, _ := os.Getwd()

	switch cmd {
	case "start":
		fs := flag.NewFlagSet("start", flag.ExitOnError)
		interfaces := fs.String("interfaces", "", "comma-separated interface ids")
		featureID := fs.String("feature-id", "", "feature id for long-running stability")
		syncLevel := fs.String("sync-level", "", "sync level (0|1|2)")
		mapTokens := fs.Int("map-tokens", 0, "repository map token budget")
		mapRefresh := fs.Int("map-refresh", 0, "repository map refresh seconds")
		_ = fs.Parse(os.Args[2:])
		res, err := core.Start(cwd, ad, core.StartFlags{Interfaces: *interfaces, FeatureID: *featureID, SyncLevel: *syncLevel, MapTokens: *mapTokens, MapRefresh: *mapRefresh})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "blocked:") {
				os.Exit(6)
			}
			os.Exit(1)
		}
		if code := deferredOutcomeExitCode(res.Step); code != 0 {
			emitDeferredOutcome(res.Step)
			os.Exit(code)
		}
		maybeScheduleAutoGC(cwd, res.Step)
	case "doctor":
		handleDoctor(cwd, ad)
	case "gc":
		handleGC(cwd)
	case "wrap":
		handleWrap()
	case "hook":
		handleHook()
	case "ide-autostart":
		handleIDEAutoStart(cwd)
	case "init":
		handleInit(cwd, ad)
	case "migrate":
		handleMigrate(cwd)
	case "resume":
		handleResume(cwd, ad)
	default:
		usage()
		os.Exit(2)
	}
}

func handleDoctor(cwd string, ad adapter.CLIAdapter) {
	startedAt := time.Now()
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	syncProposal := fs.Bool("sync-proposal", false, "show AGENTS->contract proposal")
	applySync := fs.Bool("apply-sync", false, "apply sync proposal (approval required)")
	syncLevel := fs.String("sync-level", "", "sync level (0|1|2)")
	_ = fs.Parse(os.Args[2:])

	src, created, err := bootstrap.EnsureRootAGENTS(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize AGENTS.md:", err)
		os.Exit(1)
	}
	if created {
		fmt.Println("created AGENTS.md at repo root")
	}

	level := syncpolicy.ParseLevel(*syncLevel)
	if *syncLevel == "" {
		level = syncpolicy.ParseLevel(os.Getenv("ATRAKTA_SYNC_LEVEL"))
	}
	if *syncProposal || level == syncpolicy.Level1 {
		c, _, err := contract.LoadOrInit(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to load contract:", err)
			os.Exit(1)
		}
		resolvedSrc, _, err := agentsctx.Resolve(agentsctx.ResolveInput{
			RepoRoot: cwd,
			StartDir: cwd,
			Config:   c.Context,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to resolve AGENTS context:", err)
			os.Exit(1)
		}
		src = resolvedSrc
		if level == syncpolicy.Level2 {
			fmt.Println("sync proposal disabled in level 2 (strict mode)")
		} else {
			sp, proposed, err := syncpolicy.ProposeFromAGENTS(c, src)
			if err != nil {
				fmt.Fprintln(os.Stderr, "sync proposal failed:", err)
				os.Exit(1)
			}
			b, _ := json.MarshalIndent(sp, "", "  ")
			fmt.Println(string(b))
			if sp.Needed && *applySync {
				resp := ad.RequestApproval(map[string]any{"summary": sp.Summary, "proposal": sp})
				if !resp.Approved {
					fmt.Println("sync proposal not approved")
				} else {
					if _, err := contract.Save(cwd, proposed); err != nil {
						fmt.Fprintln(os.Stderr, "failed to apply sync proposal:", err)
						os.Exit(1)
					}
					fmt.Println("sync proposal applied to .atrakta/contract.json")
				}
			}
		}
	}
	report, _, err := doctor.Run(cwd, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctor: %s\n", report.Reason)
		os.Exit(1)
	}
	fmt.Printf("doctor: %s\n", report.Reason)
	printSelfHealProposals(cwd)
	if snap, err := runtimeobs.Record(cwd, "doctor", time.Since(startedAt)); err == nil {
		fmt.Printf("runtime metrics: doctor last=%dms p95=%dms n=%d\n", snap.LastMs, snap.P95Ms, snap.Count)
	}
}

func handleGC(cwd string) {
	fs := flag.NewFlagSet("gc", flag.ExitOnError)
	scopeRaw := fs.String("scope", "tmp,events", "comma-separated scopes: tmp,events")
	apply := fs.Bool("apply", false, "apply deletion for supported scopes")
	auto := fs.Bool("auto", false, "auto mode (threshold-triggered)")
	_ = fs.Parse(os.Args[2:])

	scopes, err := parseGCScopes(*scopeRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	rep, err := gcengine.Run(gcengine.Request{
		RepoRoot: cwd,
		Scopes:   scopes,
		Apply:    *apply,
		Auto:     *auto,
	}, gcengine.DefaultConfig())
	if err != nil {
		fmt.Fprintln(os.Stderr, "gc failed:", err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(rep, "", "  ")
	fmt.Println(string(b))
}

func handleIDEAutoStart(cwd string) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: atrakta ide-autostart [install|uninstall|status]")
		os.Exit(2)
	}
	switch os.Args[2] {
	case "install":
		changed, path, err := ide.Install(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if changed {
			fmt.Printf("ide autostart installed: %s\n", path)
		} else {
			fmt.Printf("ide autostart already installed: %s\n", path)
		}
	case "uninstall":
		changed, path, err := ide.Uninstall(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if changed {
			fmt.Printf("ide autostart removed: %s\n", path)
		} else {
			fmt.Printf("ide autostart not present: %s\n", path)
		}
	case "status":
		st := ide.Check(cwd)
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(b))
	default:
		fmt.Fprintln(os.Stderr, "usage: atrakta ide-autostart [install|uninstall|status]")
		os.Exit(2)
	}
}

func handleInit(cwd string, ad adapter.CLIAdapter) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	interfaces := fs.String("interfaces", "", "comma-separated interface ids")
	featureID := fs.String("feature-id", "", "feature id for long-running stability")
	syncLevel := fs.String("sync-level", "", "sync level (0|1|2)")
	mapTokens := fs.Int("map-tokens", 0, "repository map token budget")
	mapRefresh := fs.Int("map-refresh", 0, "repository map refresh seconds")
	noHook := fs.Bool("no-hook", false, "skip hook install")
	_ = fs.Parse(os.Args[2:])

	self, _ := os.Executable()
	if err := wrapper.Install(self); err != nil {
		fmt.Fprintln(os.Stderr, "init failed at wrap install:", err)
		os.Exit(1)
	}
	if !*noHook {
		if err := hooks.Install(self); err != nil {
			fmt.Fprintln(os.Stderr, "init failed at hook install:", err)
			os.Exit(1)
		}
	}
	if changed, path, err := ide.Install(cwd); err != nil {
		fmt.Fprintln(os.Stderr, "init failed at ide autostart install:", err)
		os.Exit(1)
	} else if changed {
		fmt.Printf("ide autostart installed: %s\n", path)
	}

	res, err := core.Start(cwd, ad, core.StartFlags{Interfaces: *interfaces, FeatureID: *featureID, SyncLevel: *syncLevel, MapTokens: *mapTokens, MapRefresh: *mapRefresh})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "blocked:") {
			os.Exit(6)
		}
		os.Exit(1)
	}
	if code := deferredOutcomeExitCode(res.Step); code != 0 {
		emitDeferredOutcome(res.Step)
		os.Exit(code)
	}
	maybeScheduleAutoGC(cwd, res.Step)
}

func handleWrap() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: atrakta wrap [install|uninstall|run]")
		os.Exit(2)
	}
	sub := os.Args[2]
	self, _ := os.Executable()
	switch sub {
	case "install":
		if err := wrapper.Install(self); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "uninstall":
		if err := wrapper.Uninstall(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "run":
		fs := flag.NewFlagSet("wrap run", flag.ExitOnError)
		iface := fs.String("interface", "", "interface id")
		real := fs.String("real", "", "real executable")
		_ = fs.Parse(os.Args[3:])
		if *iface == "" {
			fmt.Fprintln(os.Stderr, "wrap run requires --interface")
			os.Exit(2)
		}
		exitCode := wrapper.Run(self, *iface, *real, fs.Args())
		os.Exit(exitCode)
	default:
		fmt.Fprintln(os.Stderr, "usage: atrakta wrap [install|uninstall|run]")
		os.Exit(2)
	}
}

func handleHook() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: atrakta hook [install|uninstall]")
		os.Exit(2)
	}
	self, _ := os.Executable()
	switch os.Args[2] {
	case "install":
		if err := hooks.Install(self); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "uninstall":
		if err := hooks.Uninstall(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: atrakta hook [install|uninstall]")
		os.Exit(2)
	}
}

func handleMigrate(cwd string) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: atrakta migrate check")
		os.Exit(2)
	}
	switch os.Args[2] {
	case "check":
		if err := migrate.Check(cwd); err != nil {
			fmt.Fprintln(os.Stderr, "migrate check failed:", err)
			os.Exit(1)
		}
		fmt.Println("migrate check: ok")
	default:
		fmt.Fprintln(os.Stderr, "usage: atrakta migrate check")
		os.Exit(2)
	}
}

func handleResume(cwd string, ad adapter.CLIAdapter) {
	fs := flag.NewFlagSet("resume", flag.ExitOnError)
	interfaces := fs.String("interfaces", "", "comma-separated interface ids (override checkpoint)")
	featureID := fs.String("feature-id", "", "feature id (override checkpoint)")
	syncLevel := fs.String("sync-level", "", "sync level (override checkpoint)")
	mapTokens := fs.Int("map-tokens", 0, "repository map token budget")
	mapRefresh := fs.Int("map-refresh", 0, "repository map refresh seconds")
	_ = fs.Parse(os.Args[2:])

	cp, err := checkpoint.LoadLatest(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "resume failed: no valid run checkpoint (run `atrakta start` first):", err)
		os.Exit(1)
	}
	flags := core.StartFlags{
		Interfaces: cp.Interfaces,
		FeatureID:  cp.FeatureID,
		SyncLevel:  cp.SyncLevel,
		MapTokens:  *mapTokens,
		MapRefresh: *mapRefresh,
	}
	if *interfaces != "" {
		flags.Interfaces = *interfaces
	}
	if *featureID != "" {
		flags.FeatureID = *featureID
	}
	if *syncLevel != "" {
		flags.SyncLevel = *syncLevel
	}
	fmt.Printf("resume: stage=%s outcome=%s feature=%s\n", cp.Stage, cp.Outcome, cp.FeatureID)
	res, err := core.Start(cwd, ad, flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "blocked:") {
			os.Exit(6)
		}
		os.Exit(1)
	}
	if code := deferredOutcomeExitCode(res.Step); code != 0 {
		emitDeferredOutcome(res.Step)
		os.Exit(code)
	}
	maybeScheduleAutoGC(cwd, res.Step)
}

// cmdName returns the invoked command name (supports "atr" alias via symlink).
func cmdName() string {
	name := filepath.Base(os.Args[0])
	// strip .exe suffix on Windows
	name = strings.TrimSuffix(name, ".exe")
	if name == "atr" {
		return "atr"
	}
	return "atrakta"
}

func usage() {
	cmd := cmdName()
	fmt.Printf("%s commands:\n", cmd)
	fmt.Printf("  %s start [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]\n", cmd)
	fmt.Printf("  %s init [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>] [--no-hook]\n", cmd)
	fmt.Printf("  %s doctor [--sync-proposal] [--apply-sync] [--sync-level <0|1|2>]\n", cmd)
	fmt.Printf("  %s gc [--scope <tmp,events>] [--apply] [--auto]\n", cmd)
	fmt.Printf("  %s wrap install\n", cmd)
	fmt.Printf("  %s wrap uninstall\n", cmd)
	fmt.Printf("  %s wrap run --interface <id> --real <path> -- [args...]\n", cmd)
	fmt.Printf("  %s hook install\n", cmd)
	fmt.Printf("  %s hook uninstall\n", cmd)
	fmt.Printf("  %s ide-autostart [install|uninstall|status]\n", cmd)
	fmt.Printf("  %s migrate check\n", cmd)
	fmt.Printf("  %s resume [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]\n", cmd)
	if cmd == "atrakta" {
		fmt.Println("\n  tip: 'atr' is a short alias for 'atrakta'")
	}
}

func deferredOutcomeExitCode(step model.StepEvent) int {
	switch strings.ToUpper(strings.TrimSpace(step.Outcome)) {
	case "NEEDS_INPUT":
		return 4
	case "NEEDS_APPROVAL":
		return 5
	case "BLOCKED":
		return 6
	default:
		return 0
	}
}

func emitDeferredOutcome(step model.StepEvent) {
	if os.Getenv("ATRAKTA_STATUS_JSON") == "1" {
		b, _ := json.Marshal(map[string]any{
			"outcome":     step.Outcome,
			"next_action": step.NextAction,
		})
		fmt.Println(string(b))
		return
	}
	fmt.Printf("outcome: %s\n", step.Outcome)
	if step.NextAction.Command != "" {
		fmt.Printf("next: %s\n", step.NextAction.Command)
	}
}

func printSelfHealProposals(cwd string) {
	ideStatus := ide.Check(cwd)
	if !ideStatus.Installed {
		fmt.Println("proposal: atrakta ide-autostart install")
	}
	health, err := wrapper.Health()
	if err != nil {
		return
	}
	if health.WrapperScriptCount == 0 {
		fmt.Println("proposal: atrakta wrap install")
		return
	}
	if !health.PathContainsUserBin || !health.PathPrefersUserBin {
		fmt.Println("proposal: atrakta wrap install (PATH priority repair)")
	}
	if st, err := gcengine.Check(cwd, gcengine.DefaultConfig()); err == nil {
		if st.TmpOverThreshold {
			fmt.Println("proposal: atrakta gc --scope tmp --apply")
		}
		if st.EventsOverThreshold {
			fmt.Println("proposal: atrakta gc --scope events")
		}
	}
}

func maybeScheduleAutoGC(cwd string, step model.StepEvent) {
	if strings.ToUpper(strings.TrimSpace(step.Outcome)) != "DONE" {
		return
	}
	if os.Getenv("ATRAKTA_GC_DISABLE") == "1" {
		return
	}
	cfg := gcengine.DefaultConfig()
	if !gcengine.ShouldRunAuto(cwd, cfg) {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	if err := gcengine.SpawnAuto(self, cwd); err != nil {
		fmt.Fprintf(os.Stderr, "gc auto trigger skipped: %v\n", err)
	}
}

func parseGCScopes(raw string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(strings.ToLower(part))
		if id == "" {
			continue
		}
		switch id {
		case "tmp", "events":
			out[id] = true
		default:
			return nil, fmt.Errorf("unknown gc scope: %s", id)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one gc scope required")
	}
	return out, nil
}
