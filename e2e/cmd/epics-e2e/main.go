package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentepics/epics.sh/e2e/internal/harness"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	repoRoot, err := harness.FindRepoRoot(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	switch args[0] {
	case "list":
		return runList()
	case "run":
		return runScenarios(repoRoot, args[1:])
	default:
		printUsage()
		return 1
	}
}

func runList() int {
	if err := harness.PrintList(os.Stdout, harness.DefaultScenarios()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func runScenarios(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	scenarioFlag := fs.String("scenario", "", "comma-separated scenario names")
	tagFlag := fs.String("tag", "", "run only scenarios with the given tag")
	keepArtifacts := fs.Bool("keep-artifacts", false, "retain passing scenario artifacts")
	jsonOutput := fs.Bool("json", false, "emit JSON summary")
	artifactsDir := fs.String("artifacts-dir", filepath.Join(repoRoot, ".e2e-artifacts"), "artifact output directory")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	selected, err := harness.SelectScenarios(harness.DefaultScenarios(), harness.SplitList(*scenarioFlag), *tagFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	runner := harness.Runner{
		RepoRoot:      repoRoot,
		ArtifactsBase: *artifactsDir,
		KeepArtifacts: *keepArtifacts,
	}

	summary, err := runner.Run(selected)
	if *jsonOutput {
		raw, marshalErr := json.MarshalIndent(summary, "", "  ")
		if marshalErr != nil {
			fmt.Fprintln(os.Stderr, marshalErr.Error())
			return 1
		}
		raw = append(raw, '\n')
		_, _ = os.Stdout.Write(raw)
	} else {
		printSummary(summary)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if summary.FailedCount > 0 {
		return 1
	}
	return 0
}

func printSummary(summary harness.Summary) {
	fmt.Fprintf(os.Stdout, "Run ID: %s\n", summary.RunID)
	fmt.Fprintf(os.Stdout, "Image: %s\n", summary.ImageTag)
	fmt.Fprintf(os.Stdout, "Artifacts: %s\n", summary.ArtifactRoot)
	if summary.RunLogPath != "" {
		fmt.Fprintf(os.Stdout, "Run log: %s\n", summary.RunLogPath)
	}
	if summary.RunEventLogPath != "" {
		fmt.Fprintf(os.Stdout, "Run events: %s\n", summary.RunEventLogPath)
	}
	for _, result := range summary.Results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(os.Stdout, "%s %s\n", status, result.Name)
		if result.ScenarioLogPath != "" {
			fmt.Fprintf(os.Stdout, "  log: %s\n", result.ScenarioLogPath)
		}
		if result.ScenarioEventLogPath != "" {
			fmt.Fprintf(os.Stdout, "  events: %s\n", result.ScenarioEventLogPath)
		}
		if result.Error != "" {
			fmt.Fprintf(os.Stdout, "  %s\n", result.Error)
		}
	}
	fmt.Fprintf(os.Stdout, "Passed: %d Failed: %d\n", summary.PassedCount, summary.FailedCount)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: epics-e2e <command>")
	fmt.Fprintln(os.Stderr, "Commands: list, run")
}
