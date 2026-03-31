package main

import "os"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		runWithErrorHandling(cmd, os.Args[2:], runCommand)
	case "start":
		runWithErrorHandling(cmd, os.Args[2:], startCommand)
	case "init":
		runWithErrorHandling(cmd, os.Args[2:], initCommand)
	case "wrap":
		runWithErrorHandling(cmd, os.Args[2:], runWrap)
	case "hook":
		runWithErrorHandling(cmd, os.Args[2:], runHook)
	case "ide-autostart":
		runWithErrorHandling(cmd, os.Args[2:], runIDEAutostart)
	case "resume":
		runWithErrorHandling(cmd, os.Args[2:], resumeCommand)
	case "inspect", "preview", "simulate":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runMode(cmd, args) })
	case "projection":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runProjection(args) })
	case "gc":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runGC(args) })
	case "migrate":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runMigrate(args) })
	case "run-fixtures":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runFixtures(args) })
	case "accept":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runAccept(args) })
	case "mutate":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runMutate(args) })
	case "audit":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runAudit(args) })
	case "doctor", "parity", "integration":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runAlias(cmd, args) })
	case "extensions":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runExtensions(args) })
	case "onboard":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runOnboard(args) })
	case "harness-profile":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runHarnessProfile(args) })
	case "benchmark":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runBenchmark(args) })
	case "export-snapshots":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runExportSnapshots(args) })
	case "verify-coverage":
		runWithErrorHandling(cmd, os.Args[2:], func(args []string) (int, error) { return exitOK, runVerifyCoverage(args) })
	default:
		usage()
		os.Exit(2)
	}
}
