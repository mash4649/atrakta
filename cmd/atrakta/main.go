package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"atrakta/internal/adapter"
	"atrakta/internal/bootstrap"
	"atrakta/internal/brownfield"
	"atrakta/internal/checkpoint"
	agentsctx "atrakta/internal/context"
	"atrakta/internal/contract"
	"atrakta/internal/core"
	"atrakta/internal/doctor"
	gcengine "atrakta/internal/gc"
	"atrakta/internal/hooks"
	"atrakta/internal/ide"
	"atrakta/internal/manifest"
	"atrakta/internal/migrate"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/registry"
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
		handleHook(cwd)
	case "ide-autostart":
		handleIDEAutoStart(cwd)
	case "init":
		handleInit(cwd, ad)
	case "migrate":
		handleMigrate(cwd)
	case "resume":
		handleResume(cwd, ad)
	case "projection":
		handleProjection(cwd, ad)
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
	parity := fs.Bool("parity", false, "run parity drift diagnostics")
	integration := fs.Bool("integration", false, "run brownfield integration diagnostics")
	asJSON := fs.Bool("json", false, "print machine-readable output (for parity/integration mode)")
	_ = fs.Parse(os.Args[2:])
	if *parity && *integration {
		fmt.Fprintln(os.Stderr, "doctor: --parity and --integration cannot be used together")
		os.Exit(2)
	}

	if *parity {
		rep, err := doctor.RunParity(cwd)
		if *asJSON {
			fmt.Println(rep.JSON())
		} else {
			fmt.Printf("doctor --parity: %s\n", rep.Reason)
			for _, f := range rep.BlockingIssues {
				fmt.Printf("  [BLOCK] %s: %s", f.Code, f.Message)
				if f.Path != "" {
					fmt.Printf(" (%s)", f.Path)
				}
				fmt.Println()
			}
			for _, f := range rep.Warnings {
				fmt.Printf("  [WARN ] %s: %s", f.Code, f.Message)
				if f.Path != "" {
					fmt.Printf(" (%s)", f.Path)
				}
				fmt.Println()
			}
			for _, cmd := range rep.SuggestedCommands {
				fmt.Printf("  suggestion: %s\n", cmd)
			}
		}
		if err != nil || rep.Outcome == "BLOCKED" {
			os.Exit(1)
		}
		return
	}
	if *integration {
		rep, err := doctor.RunIntegration(cwd)
		if *asJSON {
			fmt.Println(rep.JSON())
		} else {
			fmt.Printf("doctor --integration: %s\n", rep.Reason)
			for _, f := range rep.BlockingIssues {
				fmt.Printf("  [BLOCK] %s: %s", f.Code, f.Message)
				if f.Path != "" {
					fmt.Printf(" (%s)", f.Path)
				}
				fmt.Println()
			}
			for _, f := range rep.Warnings {
				fmt.Printf("  [WARN ] %s: %s", f.Code, f.Message)
				if f.Path != "" {
					fmt.Printf(" (%s)", f.Path)
				}
				fmt.Println()
			}
			for _, cmd := range rep.SuggestedCommands {
				fmt.Printf("  suggestion: %s\n", cmd)
			}
		}
		if err != nil || rep.Outcome == "BLOCKED" {
			os.Exit(1)
		}
		return
	}

	c, _, err := contract.LoadOrInit(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load contract:", err)
		os.Exit(1)
	}
	agentsMode := "append"
	agentsAppendFile := ".atrakta/AGENTS.append.md"
	if c.Extensions != nil && c.Extensions.Agents != nil {
		if strings.TrimSpace(c.Extensions.Agents.Mode) != "" {
			agentsMode = c.Extensions.Agents.Mode
		}
		if strings.TrimSpace(c.Extensions.Agents.AppendFile) != "" {
			agentsAppendFile = c.Extensions.Agents.AppendFile
		}
	}
	src, created, err := bootstrap.EnsureRootAGENTSWithMode(cwd, agentsMode, agentsAppendFile)
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
	mode := fs.String("mode", "greenfield", "init mode: greenfield|brownfield")
	interfaces := fs.String("interfaces", "", "comma-separated interface ids")
	featureID := fs.String("feature-id", "", "feature id for long-running stability")
	syncLevel := fs.String("sync-level", "", "sync level (0|1|2)")
	mapTokens := fs.Int("map-tokens", 0, "repository map token budget")
	mapRefresh := fs.Int("map-refresh", 0, "repository map refresh seconds")
	mergeStrategy := fs.String("merge-strategy", "", "brownfield merge strategy: append|include|replace")
	agentsMode := fs.String("agents-mode", "", "brownfield agents mode: append|include|generate")
	noOverwrite := fs.Bool("no-overwrite", false, "brownfield: never overwrite existing user-managed files")
	noHook := fs.Bool("no-hook", false, "skip hook install")
	_ = fs.Parse(os.Args[2:])

	self, _ := os.Executable()

	resolvedMode := strings.TrimSpace(strings.ToLower(*mode))
	if resolvedMode == "" {
		resolvedMode = "greenfield"
	}
	if resolvedMode != "greenfield" && resolvedMode != "brownfield" {
		fmt.Fprintln(os.Stderr, "init: --mode must be greenfield|brownfield")
		os.Exit(2)
	}

	resolvedInterfaces := *interfaces
	if resolvedMode == "brownfield" {
		merge, agents, err := normalizeBrownfieldModes(*mergeStrategy, *agentsMode)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init:", err)
			os.Exit(2)
		}

		c, _, err := contract.LoadOrInit(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed at contract load:", err)
			os.Exit(1)
		}
		c = contract.CanonicalizeBoundary(c)
		if c.Extensions == nil {
			def := contract.Default(cwd)
			c.Extensions = def.Extensions
		}
		if c.Extensions.Agents == nil {
			c.Extensions.Agents = &contract.AgentsExtension{Mode: "append", AppendFile: ".atrakta/AGENTS.append.md"}
		}
		c.Extensions.MergeMode = merge
		c.Extensions.Agents.Mode = agents
		if strings.TrimSpace(c.Extensions.Agents.AppendFile) == "" {
			c.Extensions.Agents.AppendFile = ".atrakta/AGENTS.append.md"
		}
		cb, err := contract.Save(cwd, c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed at contract save:", err)
			os.Exit(1)
		}

		targetIDs := resolveInitInterfaces(*interfaces, c)
		resolvedInterfaces = strings.Join(targetIDs, ",")
		sourceAGENTS, _, err := bootstrap.EnsureRootAGENTSWithMode(cwd, agents, c.Extensions.Agents.AppendFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed at AGENTS preparation:", err)
			os.Exit(1)
		}
		detection, err := brownfield.Detect(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed at brownfield detection:", err)
			os.Exit(1)
		}
		reg := registry.ApplyOverrides(registry.Default(), c)
		desired, err := projection.RequiredForTargets(cwd, c, reg, targetIDs, contract.ContractHash(cb), sourceAGENTS)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed at projection planning:", err)
			os.Exit(1)
		}
		conflicts, err := brownfield.FindConflicts(cwd, desired, *noOverwrite)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed at conflict detection:", err)
			os.Exit(1)
		}
		if *noOverwrite && len(conflicts) > 0 {
			patchPath, err := brownfield.WriteProposalPatch(cwd, brownfield.ProposalInput{
				Mode:          "brownfield",
				MergeStrategy: merge,
				AgentsMode:    agents,
				NoOverwrite:   true,
				Interfaces:    targetIDs,
				Detection:     detection,
				Conflicts:     conflicts,
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "init failed at proposal generation:", err)
				os.Exit(1)
			}
			fmt.Printf("brownfield init: overwrite risk detected (%d file(s)); proposal saved: %s\n", len(conflicts), patchPath)
			fmt.Println("brownfield init completed without overwrite")
			return
		}
		if *noOverwrite {
			fmt.Println("brownfield init: --no-overwrite active; skipping wrapper/hook/ide auto-install")
		}
	}

	if !(resolvedMode == "brownfield" && *noOverwrite) {
		if err := wrapper.Install(self); err != nil {
			fmt.Fprintln(os.Stderr, "init failed at wrap install:", err)
			os.Exit(1)
		}
		if !*noHook {
			if err := hooks.InstallForRepo(cwd, self, nil); err != nil {
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
	}

	res, err := core.Start(cwd, ad, core.StartFlags{Interfaces: resolvedInterfaces, FeatureID: *featureID, SyncLevel: *syncLevel, MapTokens: *mapTokens, MapRefresh: *mapRefresh})
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

func normalizeBrownfieldModes(mergeStrategy, agentsMode string) (string, string, error) {
	merge := strings.TrimSpace(strings.ToLower(mergeStrategy))
	if merge == "" {
		merge = "append"
	}
	switch merge {
	case "append", "include", "replace":
	default:
		return "", "", fmt.Errorf("--merge-strategy must be append|include|replace")
	}

	agents := strings.TrimSpace(strings.ToLower(agentsMode))
	if agents == "" {
		switch merge {
		case "include":
			agents = "include"
		case "replace":
			agents = "generate"
		default:
			agents = "append"
		}
	}
	switch agents {
	case "append", "include", "generate":
	default:
		return "", "", fmt.Errorf("--agents-mode must be append|include|generate")
	}
	return merge, agents, nil
}

func resolveInitInterfaces(raw string, c contract.Contract) []string {
	raw = strings.TrimSpace(raw)
	sup := contract.SupportedSet(c)
	out := make([]string, 0)
	seen := map[string]struct{}{}
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := sup[id]; !ok {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			add(part)
		}
	}
	if len(out) == 0 {
		for _, id := range c.Interfaces.CoreSet {
			add(id)
		}
	}
	if len(out) == 0 {
		add("cursor")
	}
	sort.Strings(out)
	return out
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

func handleHook(cwd string) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: atrakta hook [install|uninstall|status|repair]")
		os.Exit(2)
	}
	self, _ := os.Executable()
	switch os.Args[2] {
	case "install":
		fs := flag.NewFlagSet("hook install", flag.ExitOnError)
		surface := fs.String("surface", "", "comma-separated surface ids")
		_ = fs.Parse(os.Args[3:])
		var surfaces []string
		if strings.TrimSpace(*surface) != "" {
			surfaces = []string{*surface}
		}
		if err := hooks.InstallForRepo(cwd, self, surfaces); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "uninstall":
		fs := flag.NewFlagSet("hook uninstall", flag.ExitOnError)
		surface := fs.String("surface", "", "comma-separated surface ids")
		_ = fs.Parse(os.Args[3:])
		var surfaces []string
		if strings.TrimSpace(*surface) != "" {
			surfaces = []string{*surface}
		}
		if err := hooks.UninstallForRepo(cwd, surfaces); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "status":
		fs := flag.NewFlagSet("hook status", flag.ExitOnError)
		surface := fs.String("surface", "", "comma-separated surface ids")
		asJSON := fs.Bool("json", false, "print machine-readable output")
		_ = fs.Parse(os.Args[3:])
		var surfaces []string
		if strings.TrimSpace(*surface) != "" {
			surfaces = []string{*surface}
		}
		report, err := hooks.Status(cwd, surfaces)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *asJSON {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
			return
		}
		for _, row := range report.Surfaces {
			state := "missing"
			if row.Installed {
				state = "installed"
			}
			fmt.Printf("%s: %s\n", row.Surface, state)
			for _, p := range row.Paths {
				fmt.Printf("  - %s\n", p)
			}
		}
	case "repair":
		fs := flag.NewFlagSet("hook repair", flag.ExitOnError)
		surface := fs.String("surface", "", "comma-separated surface ids")
		_ = fs.Parse(os.Args[3:])
		var surfaces []string
		if strings.TrimSpace(*surface) != "" {
			surfaces = []string{*surface}
		}
		if err := hooks.RepairForRepo(cwd, self, surfaces); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: atrakta hook [install|uninstall|status|repair]")
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

func handleProjection(cwd string, ad adapter.CLIAdapter) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: atrakta projection [render|status|repair]")
		os.Exit(2)
	}
	switch os.Args[2] {
	case "render":
		fs := flag.NewFlagSet("projection render", flag.ExitOnError)
		iface := fs.String("interface", "", "interface id")
		all := fs.Bool("all", false, "render all projection-capable interfaces")
		_ = fs.Parse(os.Args[3:])
		interfaces, err := resolveProjectionInterfaces(cwd, *iface, *all)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		res, err := core.Start(cwd, ad, core.StartFlags{Interfaces: interfaces, FeatureID: "projection-render"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "blocked:") {
				os.Exit(6)
			}
			os.Exit(1)
		}
		fmt.Printf("projection render: result=%s ops=%d\n", res.Apply.Result, len(res.Apply.Ops))
	case "status":
		fs := flag.NewFlagSet("projection status", flag.ExitOnError)
		asJSON := fs.Bool("json", false, "print machine-readable status")
		_ = fs.Parse(os.Args[3:])
		st, err := manifest.ReadStatus(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *asJSON {
			b, _ := json.MarshalIndent(st, "", "  ")
			fmt.Println(string(b))
			return
		}
		fmt.Printf("projection manifest: %s (exists=%v entries=%d)\n", st.ProjectionPath, st.ProjectionExists, len(st.Projection.Entries))
		fmt.Printf("extension manifest: %s (exists=%v entries=%d)\n", st.ExtensionPath, st.ExtensionExists, len(st.Extension.Entries))
	case "repair":
		fs := flag.NewFlagSet("projection repair", flag.ExitOnError)
		iface := fs.String("interface", "", "interface id")
		all := fs.Bool("all", false, "repair all projection-capable interfaces")
		_ = fs.Parse(os.Args[3:])
		interfaces, err := resolveProjectionInterfaces(cwd, *iface, *all)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		res, err := core.Start(cwd, ad, core.StartFlags{Interfaces: interfaces, FeatureID: "projection-repair"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "blocked:") {
				os.Exit(6)
			}
			os.Exit(1)
		}
		fmt.Printf("projection repair: result=%s ops=%d\n", res.Apply.Result, len(res.Apply.Ops))
	default:
		fmt.Fprintln(os.Stderr, "usage: atrakta projection [render|status|repair]")
		os.Exit(2)
	}
}

func resolveProjectionInterfaces(cwd, iface string, all bool) (string, error) {
	iface = strings.TrimSpace(iface)
	if iface != "" && all {
		return "", fmt.Errorf("--interface and --all are mutually exclusive")
	}
	if !all {
		return iface, nil
	}
	c, _, err := contract.LoadOrInit(cwd)
	if err != nil {
		return "", fmt.Errorf("load contract: %w", err)
	}
	reg := registry.ApplyOverrides(registry.Default(), c)
	ids := make([]string, 0, len(reg.Entries))
	for _, id := range reg.InterfaceIDs() {
		entry := reg.Entries[id]
		if strings.TrimSpace(entry.ProjectionDir) == "" {
			continue
		}
		ids = append(ids, id)
	}
	return strings.Join(ids, ","), nil
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
	fmt.Printf("  %s init [--mode <greenfield|brownfield>] [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>] [--merge-strategy <append|include|replace>] [--agents-mode <append|include|generate>] [--no-overwrite] [--no-hook]\n", cmd)
	fmt.Printf("  %s doctor [--sync-proposal] [--apply-sync] [--sync-level <0|1|2>] [--parity|--integration] [--json]\n", cmd)
	fmt.Printf("  %s gc [--scope <tmp,events>] [--apply] [--auto]\n", cmd)
	fmt.Printf("  %s wrap install\n", cmd)
	fmt.Printf("  %s wrap uninstall\n", cmd)
	fmt.Printf("  %s wrap run --interface <id> --real <path> -- [args...]\n", cmd)
	fmt.Printf("  %s hook install [--surface <surface_id,...>]\n", cmd)
	fmt.Printf("  %s hook uninstall [--surface <surface_id,...>]\n", cmd)
	fmt.Printf("  %s hook status [--surface <surface_id,...>] [--json]\n", cmd)
	fmt.Printf("  %s hook repair [--surface <surface_id,...>]\n", cmd)
	fmt.Printf("  %s ide-autostart [install|uninstall|status]\n", cmd)
	fmt.Printf("  %s migrate check\n", cmd)
	fmt.Printf("  %s resume [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]\n", cmd)
	fmt.Printf("  %s projection render [--interface <id>] [--all]\n", cmd)
	fmt.Printf("  %s projection status [--json]\n", cmd)
	fmt.Printf("  %s projection repair [--interface <id>] [--all]\n", cmd)
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
