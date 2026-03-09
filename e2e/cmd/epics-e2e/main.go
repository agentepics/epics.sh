package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	if err := loadEnvFile(filepath.Join(repoRoot, ".env.e2e")); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	switch args[0] {
	case "list":
		return runList()
	case "run":
		return runScenarios(repoRoot, args[1:])
	case "chat":
		return runChat(repoRoot, args[1:])
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
	excludeTagFlag := fs.String("exclude-tag", "", "exclude scenarios with the given tag")
	keepArtifacts := fs.Bool("keep-artifacts", false, "retain passing scenario artifacts")
	jsonOutput := fs.Bool("json", false, "emit JSON summary")
	artifactsDir := fs.String("artifacts-dir", filepath.Join(repoRoot, ".e2e-artifacts"), "artifact output directory")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	selected, err := harness.SelectScenarios(harness.DefaultScenarios(), harness.SplitList(*scenarioFlag), *tagFlag, *excludeTagFlag)
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

func runChat(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("chat", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	artifactsDir := fs.String("artifacts-dir", filepath.Join(repoRoot, ".e2e-artifacts"), "artifact output directory")
	containerName := fs.String("container-name", "", "optional Docker container name")
	workspaceFixture := fs.String("workspace-fixture", filepath.Join("e2e", "fixtures", "claude-web-project"), "workspace fixture to mount into the chat container")
	epicFixture := fs.String("epic-fixture", filepath.Join("examples", "fixtures", "resume-epic"), "Epic fixture to install before chatting")
	cleanup := fs.Bool("cleanup", false, "remove the container after the scripted chat completes")
	jsonOutput := fs.Bool("json", false, "emit JSON result")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	result, err := harness.RunLiveChat(repoRoot, harness.ChatOptions{
		ArtifactsBase:    *artifactsDir,
		ContainerName:    *containerName,
		KeepContainer:    !*cleanup,
		WorkspaceFixture: *workspaceFixture,
		EpicFixture:      *epicFixture,
	})
	if *jsonOutput {
		raw, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			fmt.Fprintln(os.Stderr, marshalErr.Error())
			return 1
		}
		raw = append(raw, '\n')
		_, _ = os.Stdout.Write(raw)
	} else {
		fmt.Fprintf(os.Stdout, "Run ID: %s\n", result.RunID)
		fmt.Fprintf(os.Stdout, "Artifacts: %s\n", result.ArtifactRoot)
		fmt.Fprintf(os.Stdout, "Image: %s\n", result.ImageTag)
		fmt.Fprintf(os.Stdout, "Container: %s\n", result.ContainerName)
		fmt.Fprintf(os.Stdout, "Transcript: %s\n", result.TranscriptPath)
		if result.HostDoctorPath != "" {
			fmt.Fprintf(os.Stdout, "Host doctor: %s\n", result.HostDoctorPath)
		}
		if result.ShellCommand != "" {
			fmt.Fprintf(os.Stdout, "Shell: %s\n", result.ShellCommand)
		}
		if result.CleanupCommand != "" {
			fmt.Fprintf(os.Stdout, "Cleanup: %s\n", result.CleanupCommand)
		} else {
			fmt.Fprintln(os.Stdout, "Cleanup: container removed automatically after chat")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func printSummary(summary harness.Summary) {
	fmt.Fprintf(os.Stdout, "Run ID: %s\n", summary.RunID)
	for _, profile := range sortedProfiles(summary.ImageTags) {
		fmt.Fprintf(os.Stdout, "Image (%s): %s\n", profile, summary.ImageTags[profile])
	}
	fmt.Fprintf(os.Stdout, "Artifacts: %s\n", summary.ArtifactRoot)
	if summary.RunLogPath != "" {
		fmt.Fprintf(os.Stdout, "Run log: %s\n", summary.RunLogPath)
	}
	if summary.RunEventLogPath != "" {
		fmt.Fprintf(os.Stdout, "Run events: %s\n", summary.RunEventLogPath)
	}
	for _, result := range summary.Results {
		status := "PASS"
		if result.Skipped {
			status = "SKIP"
		} else if !result.Passed {
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
	fmt.Fprintf(os.Stdout, "Passed: %d Failed: %d Skipped: %d\n", summary.PassedCount, summary.FailedCount, summary.SkippedCount)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: epics-e2e <command>")
	fmt.Fprintln(os.Stderr, "Commands: list, run, chat")
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid env line in %s: %q", path, line)
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			return fmt.Errorf("invalid env key in %s: %q", path, line)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func sortedProfiles(values map[string]string) []string {
	profiles := make([]string, 0, len(values))
	for profile := range values {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)
	return profiles
}
