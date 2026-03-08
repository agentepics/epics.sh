package harness

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Runner struct {
	RepoRoot      string
	ArtifactsBase string
	KeepArtifacts bool
}

type operationLogger struct {
	path      string
	eventPath string
	file      *os.File
	eventFile *os.File
}

func FindRepoRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		goMod := filepath.Join(current, "go.mod")
		mainFile := filepath.Join(current, "cmd", "epics", "main.go")
		if exists(goMod) && exists(mainFile) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("could not find repo root from %s", start)
}

func SelectScenarios(all []Scenario, names []string, tag string) ([]Scenario, error) {
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		nameSet[name] = struct{}{}
	}

	var selected []Scenario
	for _, scenario := range all {
		if len(nameSet) > 0 {
			if _, ok := nameSet[scenario.Name]; !ok {
				continue
			}
		}
		if tag != "" && !contains(scenario.Tags, tag) {
			continue
		}
		selected = append(selected, scenario)
	}

	if len(nameSet) > 0 {
		for name := range nameSet {
			found := false
			for _, scenario := range selected {
				if scenario.Name == name {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown scenario %q", name)
			}
		}
	}

	if len(selected) == 0 {
		return nil, errors.New("no scenarios matched the current filters")
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Name < selected[j].Name
	})
	return selected, nil
}

func (r Runner) Run(scenarios []Scenario) (Summary, error) {
	if err := ensureDocker(); err != nil {
		return Summary{}, err
	}

	runID := time.Now().UTC().Format("20060102T150405Z")
	artifactRoot := filepath.Join(r.ArtifactsBase, runID)
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		return Summary{}, err
	}

	runLog, err := newOperationLogger(filepath.Join(artifactRoot, "run.log"))
	if err != nil {
		return Summary{}, err
	}
	defer runLog.Close()

	imageTag := "epics-e2e:" + strings.ToLower(runID)
	summary := Summary{
		RunID:           runID,
		ImageTag:        imageTag,
		ArtifactRoot:    artifactRoot,
		RunLogPath:      runLog.path,
		RunEventLogPath: runLog.eventPath,
		ScenarioCount:   len(scenarios),
	}
	runLog.Log("INFO", "run", "start", "ok", fmt.Sprintf("starting run with %d scenario(s)", len(scenarios)))
	runLog.Log("INFO", "run", "artifacts", "ok", fmt.Sprintf("artifact root: %s", artifactRoot))
	runLog.Log("INFO", "docker", "availability-check", "ok", "docker CLI and daemon are available")

	buildLogPath := filepath.Join(artifactRoot, "build.log")
	runLog.Log("INFO", "docker", "build-image", "start", fmt.Sprintf("building image %s using %s", imageTag, filepath.ToSlash(filepath.Join("e2e", "docker", "cli-runner.Dockerfile"))))
	if err := buildImage(r.RepoRoot, imageTag, buildLogPath); err != nil {
		runLog.Log("ERROR", "docker", "build-image", "fail", err.Error())
		_ = writeSummary(artifactRoot, summary)
		return summary, err
	}
	runLog.Log("INFO", "docker", "build-image", "ok", fmt.Sprintf("image %s built successfully; build log: %s", imageTag, buildLogPath))
	defer removeImage(imageTag)

	for _, scenario := range scenarios {
		runLog.Log("INFO", "scenario", scenario.Name, "start", scenario.Description)
		result := r.runScenario(imageTag, artifactRoot, scenario, runLog)
		if result.Passed {
			summary.PassedCount++
			runLog.Log("INFO", "scenario", scenario.Name, "ok", fmt.Sprintf("scenario passed; log: %s", result.ScenarioLogPath))
		} else {
			summary.FailedCount++
			runLog.Log("ERROR", "scenario", scenario.Name, "fail", fmt.Sprintf("%s; log: %s", result.Error, result.ScenarioLogPath))
		}
		summary.Results = append(summary.Results, result)
	}

	if err := writeSummary(artifactRoot, summary); err != nil {
		runLog.Log("ERROR", "run", "write-summary", "fail", err.Error())
		return summary, err
	}
	runLog.Log("INFO", "run", "write-summary", "ok", filepath.Join(artifactRoot, "summary.json"))
	runLog.Log("INFO", "run", "complete", "ok", fmt.Sprintf("passed=%d failed=%d", summary.PassedCount, summary.FailedCount))
	return summary, nil
}

func (r Runner) runScenario(imageTag, artifactRoot string, scenario Scenario, runLog *operationLogger) ScenarioResult {
	scenarioDir := filepath.Join(artifactRoot, sanitizeName(scenario.Name))
	workspaceDir := filepath.Join(scenarioDir, "workspace")
	result := ScenarioResult{
		Name:                 scenario.Name,
		ArtifactDir:          scenarioDir,
		ScenarioLogPath:      filepath.Join(scenarioDir, "operations.log"),
		ScenarioEventLogPath: eventLogPath(filepath.Join(scenarioDir, "operations.log")),
	}

	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		result.Error = err.Error()
		return result
	}

	scenarioLog, err := newOperationLogger(result.ScenarioLogPath)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer scenarioLog.Close()

	scenarioLog.Log("INFO", scenario.Name, "start", "ok", scenario.Description)
	scenarioLog.Log("INFO", scenario.Name, "workspace", "ok", fmt.Sprintf("workspace: %s", workspaceDir))
	result.InitialManifestPath = filepath.Join(scenarioDir, "workspace.initial.manifest.json")
	if _, err := snapshotWorkspace(workspaceDir, result.InitialManifestPath); err != nil {
		result.Error = err.Error()
		scenarioLog.Log("ERROR", scenario.Name, "manifest-initial", "fail", err.Error())
		return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
	}
	scenarioLog.Log("INFO", scenario.Name, "manifest-initial", "ok", result.InitialManifestPath)

	if err := prepareWorkspace(r.RepoRoot, workspaceDir, scenario, scenarioLog); err != nil {
		result.Error = err.Error()
		scenarioLog.Log("ERROR", scenario.Name, "prepare-workspace", "fail", err.Error())
		return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
	}
	scenarioLog.Log("INFO", scenario.Name, "prepare-workspace", "ok", "workspace prepared")
	result.PreparedManifestPath = filepath.Join(scenarioDir, "workspace.prepared.manifest.json")
	if _, err := snapshotWorkspace(workspaceDir, result.PreparedManifestPath); err != nil {
		result.Error = err.Error()
		scenarioLog.Log("ERROR", scenario.Name, "manifest-prepared", "fail", err.Error())
		return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
	}
	scenarioLog.Log("INFO", scenario.Name, "manifest-prepared", "ok", result.PreparedManifestPath)

	for index, step := range scenario.Steps {
		stepResult := runStep(imageTag, workspaceDir, scenarioDir, index, step, scenarioLog)
		result.Steps = append(result.Steps, stepResult)
		writeStepArtifacts(scenarioDir, index, stepResult)
		if !stepResult.Passed {
			result.Error = stepResult.Error
			result.Passed = false
			scenarioLog.Log("ERROR", scenario.Name, "step", "fail", fmt.Sprintf("%s failed: %s", step.Name, stepResult.Error))
			return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
		}
		scenarioLog.Log("INFO", scenario.Name, "step", "ok", fmt.Sprintf("%s passed", step.Name))
	}

	if err := assertWorkspace(workspaceDir, scenario.Files, scenarioLog, scenario.Name); err != nil {
		result.Error = err.Error()
		result.Passed = false
		result.FinalManifestPath = filepath.Join(scenarioDir, "workspace.final.manifest.json")
		_, _ = snapshotWorkspace(workspaceDir, result.FinalManifestPath)
		scenarioLog.Log("ERROR", scenario.Name, "workspace-assertions", "fail", err.Error())
		return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
	}
	scenarioLog.Log("INFO", scenario.Name, "workspace-assertions", "ok", "all file assertions passed")
	result.FinalManifestPath = filepath.Join(scenarioDir, "workspace.final.manifest.json")
	if _, err := snapshotWorkspace(workspaceDir, result.FinalManifestPath); err != nil {
		result.Error = err.Error()
		result.Passed = false
		scenarioLog.Log("ERROR", scenario.Name, "manifest-final", "fail", err.Error())
		return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
	}
	scenarioLog.Log("INFO", scenario.Name, "manifest-final", "ok", result.FinalManifestPath)

	result.Passed = true
	scenarioLog.Log("INFO", scenario.Name, "complete", "ok", "scenario completed successfully")
	runLog.Log("INFO", "scenario:"+scenario.Name, "artifact-dir", "ok", scenarioDir)
	return finalizeScenarioArtifacts(r.KeepArtifacts, scenarioDir, result)
}

func finalizeScenarioArtifacts(keep bool, scenarioDir string, result ScenarioResult) ScenarioResult {
	if result.Passed && !keep {
		_ = os.RemoveAll(filepath.Join(scenarioDir, "workspace"))
	}
	return result
}

func prepareWorkspace(repoRoot, workspaceDir string, scenario Scenario, log *operationLogger) error {
	for _, copySpec := range scenario.Copies {
		src := filepath.Join(repoRoot, filepath.FromSlash(copySpec.From))
		dest := filepath.Join(workspaceDir, filepath.FromSlash(copySpec.To))
		log.Log("INFO", scenario.Name, "copy", "start", fmt.Sprintf("%s -> %s", src, dest))
		if err := copyPath(src, dest); err != nil {
			return err
		}
		log.Log("INFO", scenario.Name, "copy", "ok", fmt.Sprintf("%s -> %s", src, dest))
	}

	for relative, content := range scenario.SeedFiles {
		path := filepath.Join(workspaceDir, filepath.FromSlash(relative))
		log.Log("INFO", scenario.Name, "seed-file", "start", path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
		log.Log("INFO", scenario.Name, "seed-file", "ok", path)
	}

	return nil
}

func runStep(imageTag, workspaceDir, scenarioDir string, index int, step Step, log *operationLogger) StepResult {
	command := append([]string{"epics"}, step.Args...)
	dockerCommand := []string{"docker", "run", "--rm", "-v", workspaceDir + ":/workspace", "-w", "/workspace"}
	if step.Stdin != "" {
		dockerCommand = append(dockerCommand, "-i")
	}
	if uid, gid, ok := currentUserIDs(); ok {
		dockerCommand = append(dockerCommand, "--user", uid+":"+gid)
	}
	for _, key := range sortedEnvKeys(step.Env) {
		dockerCommand = append(dockerCommand, "-e", key+"="+step.Env[key])
	}
	dockerCommand = append(dockerCommand, imageTag)
	dockerCommand = append(dockerCommand, step.Args...)

	cmd := exec.Command(dockerCommand[0], dockerCommand[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if step.Stdin != "" {
		cmd.Stdin = strings.NewReader(step.Stdin)
	}
	startedAt := time.Now().UTC()
	stepLogPath := filepath.Join(scenarioDir, fmt.Sprintf("step-%02d-%s.operations.log", index+1, sanitizeName(step.Name)))
	beforeManifestPath := filepath.Join(scenarioDir, fmt.Sprintf("step-%02d-%s.before.manifest.json", index+1, sanitizeName(step.Name)))
	afterManifestPath := filepath.Join(scenarioDir, fmt.Sprintf("step-%02d-%s.after.manifest.json", index+1, sanitizeName(step.Name)))
	stepLog, err := newOperationLogger(stepLogPath)
	if err == nil {
		defer stepLog.Close()
		stepLog.Log("INFO", "step:"+step.Name, "start", "ok", fmt.Sprintf("starting step %s", step.Name))
		stepLog.Log("INFO", "step:"+step.Name, "command", "ok", strings.Join(command, " "))
		stepLog.Log("INFO", "step:"+step.Name, "docker-command", "ok", strings.Join(dockerCommand, " "))
		if len(step.Env) > 0 {
			stepLog.Log("INFO", "step:"+step.Name, "env", "ok", fmt.Sprintf("env=%v", step.Env))
		}
		if step.Stdin != "" {
			stepLog.Log("INFO", "step:"+step.Name, "stdin", "ok", fmt.Sprintf("stdin_preview=%q", preview(step.Stdin)))
		}
	}
	if _, err := snapshotWorkspace(workspaceDir, beforeManifestPath); err == nil {
		if stepLog != nil {
			stepLog.Log("INFO", "step:"+step.Name, "manifest-before", "ok", beforeManifestPath)
		}
		log.Log("INFO", "step:"+step.Name, "manifest-before", "ok", beforeManifestPath)
	}
	log.Log("INFO", "step:"+step.Name, "docker-run", "start", fmt.Sprintf("command=%s", strings.Join(command, " ")))

	exitCode := 0
	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	result := StepResult{
		Name:               step.Name,
		Command:            command,
		ExitCode:           exitCode,
		Stdout:             stdout.String(),
		Stderr:             stderr.String(),
		Passed:             true,
		LogPath:            stepLogPath,
		EventLogPath:       eventLogPath(stepLogPath),
		BeforeManifestPath: beforeManifestPath,
		AfterManifestPath:  afterManifestPath,
		StartedAt:          startedAt.Format(time.RFC3339),
	}
	endedAt := time.Now().UTC()
	result.EndedAt = endedAt.Format(time.RFC3339)
	result.DurationMillis = endedAt.Sub(startedAt).Milliseconds()
	if _, err := snapshotWorkspace(workspaceDir, afterManifestPath); err == nil {
		if stepLog != nil {
			stepLog.Log("INFO", "step:"+step.Name, "manifest-after", "ok", afterManifestPath)
		}
		log.Log("INFO", "step:"+step.Name, "manifest-after", "ok", afterManifestPath)
	}
	exitMessage := fmt.Sprintf("expected exit code=%d actual exit code=%d duration_ms=%d", step.ExpectExitCode, exitCode, result.DurationMillis)

	if exitCode != step.ExpectExitCode {
		result.Passed = false
		result.Error = exitMessage
		if stepLog != nil {
			stepLog.Log("ERROR", "step:"+step.Name, "docker-run", "fail", result.Error)
		}
		log.Log("ERROR", "step:"+step.Name, "docker-run", "fail", result.Error)
		return result
	}
	log.Log("INFO", "step:"+step.Name, "docker-run", "ok", exitMessage)
	if stepLog != nil {
		stepLog.Log("INFO", "step:"+step.Name, "docker-run", "ok", exitMessage)
		stepLog.Log("INFO", "step:"+step.Name, "stdout-log", "ok", fmt.Sprintf("step-%02d-%s.stdout.log", index+1, sanitizeName(step.Name)))
		stepLog.Log("INFO", "step:"+step.Name, "stderr-log", "ok", fmt.Sprintf("step-%02d-%s.stderr.log", index+1, sanitizeName(step.Name)))
	}

	if err := assertOutput(step, result.Stdout, result.Stderr, log, stepLog, step.Name); err != nil {
		result.Passed = false
		result.Error = err.Error()
		if stepLog != nil {
			stepLog.Log("ERROR", "step:"+step.Name, "assert-output", "fail", err.Error())
		}
		log.Log("ERROR", "step:"+step.Name, "assert-output", "fail", err.Error())
		return result
	}
	if stepLog != nil {
		stepLog.Log("INFO", "step:"+step.Name, "assert-output", "ok", "all stdout/stderr assertions passed")
		stepLog.Log("INFO", "step:"+step.Name, "complete", "ok", "step completed successfully")
	}
	log.Log("INFO", "step:"+step.Name, "assert-output", "ok", "all stdout/stderr assertions passed")

	return result
}

func sortedEnvKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertOutput(step Step, stdout, stderr string, scenarioLog, stepLog *operationLogger, stepName string) error {
	for _, needle := range step.StdoutContains {
		if !strings.Contains(stdout, needle) {
			return fmt.Errorf("stdout did not contain expected substring %q; actual preview=%q", needle, preview(stdout))
		}
		logAssertion(stepLog, scenarioLog, stepName, "stdout-contains", fmt.Sprintf("expected=%q actual_preview=%q", needle, preview(stdout)))
	}
	for _, needle := range step.StdoutNotContains {
		if strings.Contains(stdout, needle) {
			return fmt.Errorf("stdout unexpectedly contained forbidden substring %q; actual preview=%q", needle, preview(stdout))
		}
		logAssertion(stepLog, scenarioLog, stepName, "stdout-not-contains", fmt.Sprintf("forbidden=%q actual_preview=%q", needle, preview(stdout)))
	}
	for _, needle := range step.StderrContains {
		if !strings.Contains(stderr, needle) {
			return fmt.Errorf("stderr did not contain expected substring %q; actual preview=%q", needle, preview(stderr))
		}
		logAssertion(stepLog, scenarioLog, stepName, "stderr-contains", fmt.Sprintf("expected=%q actual_preview=%q", needle, preview(stderr)))
	}
	for _, needle := range step.StderrNotContains {
		if strings.Contains(stderr, needle) {
			return fmt.Errorf("stderr unexpectedly contained forbidden substring %q; actual preview=%q", needle, preview(stderr))
		}
		logAssertion(stepLog, scenarioLog, stepName, "stderr-not-contains", fmt.Sprintf("forbidden=%q actual_preview=%q", needle, preview(stderr)))
	}
	return nil
}

func assertWorkspace(workspaceDir string, assertions []FileAssertion, log *operationLogger, scenarioName string) error {
	for _, assertion := range assertions {
		path := filepath.Join(workspaceDir, filepath.FromSlash(assertion.Path))
		log.Log("INFO", scenarioName, "assert-file", "start", assertion.Path)
		info, err := os.Stat(path)
		if assertion.MustExist {
			if err != nil {
				return fmt.Errorf("expected %s to exist: %w", assertion.Path, err)
			}
			if info.IsDir() && (len(assertion.Contains) > 0 || len(assertion.NotContains) > 0 || assertion.Equals != "") {
				return fmt.Errorf("cannot assert file contents for directory %s", assertion.Path)
			}
			log.Log("INFO", scenarioName, "assert-file-exists", "ok", assertion.Path)
		}
		if !assertion.MustExist {
			continue
		}

		if info.IsDir() {
			continue
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(raw)
		if assertion.Equals != "" && content != assertion.Equals {
			return fmt.Errorf("expected %s to equal %q; actual preview=%q", assertion.Path, assertion.Equals, preview(content))
		}
		if assertion.Equals != "" {
			log.Log("INFO", scenarioName, "assert-file-equals", "ok", fmt.Sprintf("%s expected=%q actual_preview=%q", assertion.Path, assertion.Equals, preview(content)))
		}
		for _, needle := range assertion.Contains {
			if !strings.Contains(content, needle) {
				return fmt.Errorf("%s did not contain expected substring %q; actual preview=%q", assertion.Path, needle, preview(content))
			}
			log.Log("INFO", scenarioName, "assert-file-contains", "ok", fmt.Sprintf("%s expected=%q actual_preview=%q", assertion.Path, needle, preview(content)))
		}
		for _, needle := range assertion.NotContains {
			if strings.Contains(content, needle) {
				return fmt.Errorf("%s unexpectedly contained forbidden substring %q; actual preview=%q", assertion.Path, needle, preview(content))
			}
			log.Log("INFO", scenarioName, "assert-file-not-contains", "ok", fmt.Sprintf("%s forbidden=%q actual_preview=%q", assertion.Path, needle, preview(content)))
		}
	}
	return nil
}

func buildImage(repoRoot, imageTag, buildLogPath string) error {
	args := []string{"build", "-f", filepath.Join("e2e", "docker", "cli-runner.Dockerfile"), "-t", imageTag, "."}
	cmd := exec.Command("docker", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if writeErr := os.WriteFile(buildLogPath, output, 0o644); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}
	return nil
}

func removeImage(imageTag string) {
	cmd := exec.Command("docker", "image", "rm", "-f", imageTag)
	_ = cmd.Run()
}

func ensureDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("docker CLI was not found in PATH")
	}
	cmd := exec.Command("docker", "info")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker daemon is unavailable: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func currentUserIDs() (string, string, bool) {
	uid := strings.TrimSpace(string(mustCombinedOutput("id", "-u")))
	gid := strings.TrimSpace(string(mustCombinedOutput("id", "-g")))
	if uid == "" || gid == "" {
		return "", "", false
	}
	return uid, gid, true
}

func mustCombinedOutput(name string, args ...string) []byte {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return nil
	}
	return output
}

func writeSummary(artifactRoot string, summary Summary) error {
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(filepath.Join(artifactRoot, "summary.json"), raw, 0o644)
}

func writeStepArtifacts(scenarioDir string, index int, result StepResult) {
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		return
	}
	base := fmt.Sprintf("step-%02d-%s", index+1, sanitizeName(result.Name))
	_ = os.WriteFile(filepath.Join(scenarioDir, base+".stdout.log"), []byte(result.Stdout), 0o644)
	_ = os.WriteFile(filepath.Join(scenarioDir, base+".stderr.log"), []byte(result.Stderr), 0o644)
}

func newOperationLogger(path string) (*operationLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	eventPath := eventLogPath(path)
	eventFile, err := os.OpenFile(eventPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &operationLogger{path: path, eventPath: eventPath, file: file, eventFile: eventFile}, nil
}

func (l *operationLogger) Log(level, scope, action, status, message string) {
	if l == nil || l.file == nil {
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	_, _ = fmt.Fprintf(l.file, "%s level=%s scope=%s action=%s status=%s message=%q\n", timestamp, level, scope, action, status, message)
	if l.eventFile != nil {
		event := map[string]string{
			"timestamp": timestamp,
			"level":     level,
			"scope":     scope,
			"action":    action,
			"status":    status,
			"message":   message,
		}
		raw, err := json.Marshal(event)
		if err == nil {
			_, _ = l.eventFile.Write(append(raw, '\n'))
		}
	}
}

func (l *operationLogger) Close() error {
	if l == nil {
		return nil
	}
	var closeErr error
	if l.file != nil {
		closeErr = l.file.Close()
	}
	if l.eventFile != nil {
		if err := l.eventFile.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func logAssertion(stepLog, scenarioLog *operationLogger, stepName, kind, detail string) {
	message := fmt.Sprintf("%s %q", kind, detail)
	if stepLog != nil {
		stepLog.Log("INFO", "step:"+stepName, "assert", "ok", message)
	}
	if scenarioLog != nil {
		scenarioLog.Log("INFO", "step:"+stepName, "assert", "ok", message)
	}
}

func eventLogPath(path string) string {
	if strings.HasSuffix(path, ".log") {
		return strings.TrimSuffix(path, ".log") + ".events.jsonl"
	}
	return path + ".events.jsonl"
}

func snapshotWorkspace(workspaceDir, outputPath string) (WorkspaceManifest, error) {
	manifest := WorkspaceManifest{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Root:        workspaceDir,
		Entries:     []WorkspaceManifestEntry{},
	}
	err := filepath.WalkDir(workspaceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == workspaceDir {
			return nil
		}
		rel, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entry := WorkspaceManifestEntry{
			Path:  filepath.ToSlash(rel),
			IsDir: d.IsDir(),
			Mode:  info.Mode().String(),
			Size:  info.Size(),
		}
		if !d.IsDir() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(raw)
			entry.SHA256 = fmt.Sprintf("%x", sum[:])
		}
		manifest.Entries = append(manifest.Entries, entry)
		return nil
	})
	if err != nil {
		return WorkspaceManifest{}, err
	}
	sort.Slice(manifest.Entries, func(i, j int) bool {
		return manifest.Entries[i].Path < manifest.Entries[j].Path
	})
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return WorkspaceManifest{}, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(outputPath, raw, 0o644); err != nil {
		return WorkspaceManifest{}, err
	}
	return manifest, nil
}

func preview(value string) string {
	trimmed := strings.ReplaceAll(value, "\r\n", "\n")
	trimmed = strings.ReplaceAll(trimmed, "\n", "\\n")
	if len(trimmed) > 160 {
		return trimmed[:160] + "...(truncated)"
	}
	return trimmed
}

func copyPath(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dest, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			entryInfo, err := d.Info()
			if err != nil {
				return err
			}
			return copyFile(path, target, entryInfo.Mode())
		})
	}
	return copyFile(src, dest, info.Mode())
}

func copyFile(src, dest string, mode fs.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, raw, mode.Perm())
}

func sanitizeName(value string) string {
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-", ".", "-")
	value = replacer.Replace(strings.ToLower(value))
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func SplitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func PrintList(w io.Writer, scenarios []Scenario) error {
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].Name < scenarios[j].Name
	})
	for _, scenario := range scenarios {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", scenario.Name, strings.Join(scenario.Tags, ",")); err != nil {
			return err
		}
	}
	return nil
}
