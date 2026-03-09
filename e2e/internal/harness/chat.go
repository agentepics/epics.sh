package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ChatOptions struct {
	ArtifactsBase    string
	ContainerName    string
	KeepContainer    bool
	WorkspaceFixture string
	EpicFixture      string
	Prompts          []string
}

type ChatTurn struct {
	Prompt   string `json:"prompt"`
	Response string `json:"response"`
}

type ChatResult struct {
	RunID             string     `json:"runId"`
	ArtifactRoot      string     `json:"artifactRoot"`
	LogPath           string     `json:"logPath,omitempty"`
	EventLogPath      string     `json:"eventLogPath,omitempty"`
	ImageTag          string     `json:"imageTag"`
	ContainerName     string     `json:"containerName"`
	WorkspaceDir      string     `json:"workspaceDir"`
	TranscriptPath    string     `json:"transcriptPath"`
	HostDoctorPath    string     `json:"hostDoctorPath,omitempty"`
	ShellCommand      string     `json:"shellCommand"`
	CleanupCommand    string     `json:"cleanupCommand"`
	InstallOutputPath string     `json:"installOutputPath,omitempty"`
	Turns             []ChatTurn `json:"turns,omitempty"`
}

func RunLiveChat(repoRoot string, opts ChatOptions) (ChatResult, error) {
	if _, ok := os.LookupEnv("ANTHROPIC_API_KEY"); !ok {
		return ChatResult{}, errors.New("ANTHROPIC_API_KEY is required for live chat")
	}
	if err := ensureDocker(); err != nil {
		return ChatResult{}, err
	}
	if opts.ArtifactsBase == "" {
		opts.ArtifactsBase = filepath.Join(repoRoot, ".e2e-artifacts")
	}
	if opts.WorkspaceFixture == "" {
		opts.WorkspaceFixture = filepath.Join("e2e", "fixtures", "claude-web-project")
	}
	if opts.EpicFixture == "" {
		opts.EpicFixture = filepath.Join("examples", "fixtures", "resume-epic")
	}
	if len(opts.Prompts) == 0 {
		opts.Prompts = defaultChatPrompts()
	}

	runID := time.Now().UTC().Format("20060102T150405.000000000Z")
	artifactRoot := filepath.Join(opts.ArtifactsBase, "chat-"+runID)
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		return ChatResult{}, err
	}

	logPath := filepath.Join(artifactRoot, "chat.log")
	log, err := newOperationLogger(logPath)
	if err != nil {
		return ChatResult{}, err
	}
	defer log.Close()

	result := ChatResult{
		RunID:        runID,
		ArtifactRoot: artifactRoot,
		LogPath:      log.path,
		EventLogPath: log.eventPath,
	}

	imageTag := "epics-e2e:live-chat-claude"
	buildLogPath := filepath.Join(artifactRoot, "claude.build.log")
	log.Log("INFO", "chat", "build-image", "start", fmt.Sprintf("building image %s", imageTag))
	if err := buildImage(repoRoot, imageProfiles["claude"], imageTag, buildLogPath); err != nil {
		log.Log("ERROR", "chat", "build-image", "fail", err.Error())
		return result, err
	}
	result.ImageTag = imageTag
	log.Log("INFO", "chat", "build-image", "ok", buildLogPath)

	workspaceDir := filepath.Join(artifactRoot, "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return result, err
	}
	result.WorkspaceDir = workspaceDir

	if err := copyPath(filepath.Join(repoRoot, filepath.FromSlash(opts.WorkspaceFixture)), filepath.Join(workspaceDir, "project")); err != nil {
		return result, err
	}
	if err := copyPath(filepath.Join(repoRoot, filepath.FromSlash(opts.EpicFixture)), filepath.Join(workspaceDir, "fixtures", "resume-epic")); err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Join(workspaceDir, ".claude-home", ".config"), 0o755); err != nil {
		return result, err
	}

	containerName := opts.ContainerName
	if strings.TrimSpace(containerName) == "" {
		containerName = "epics-live-chat-" + sanitizeName(runID)
	}
	result.ContainerName = containerName
	if opts.KeepContainer {
		result.ShellCommand = "docker exec -it " + containerName + " bash"
		result.CleanupCommand = "docker rm -f " + containerName
	}

	if err := startChatContainer(containerName, imageTag, workspaceDir); err != nil {
		log.Log("ERROR", "chat", "start-container", "fail", err.Error())
		return result, err
	}
	log.Log("INFO", "chat", "start-container", "ok", containerName)
	if !opts.KeepContainer {
		defer removeContainer(containerName)
	}

	installOutput, err := dockerExec(containerName, "/workspace/project", nil, []string{
		"epics", "install", "--host", "claude", "../fixtures/resume-epic",
	})
	if err != nil {
		log.Log("ERROR", "chat", "install-epic", "fail", err.Error())
		return result, err
	}
	result.InstallOutputPath = filepath.Join(artifactRoot, "install.stdout.log")
	if err := os.WriteFile(result.InstallOutputPath, []byte(installOutput), 0o644); err != nil {
		return result, err
	}
	log.Log("INFO", "chat", "install-epic", "ok", result.InstallOutputPath)

	hostDoctorOutput, err := dockerExec(containerName, "/workspace/project", nil, []string{
		"epics", "--json", "host", "doctor", "claude",
	})
	if err != nil {
		log.Log("ERROR", "chat", "host-doctor", "fail", err.Error())
		return result, err
	}
	result.HostDoctorPath = filepath.Join(artifactRoot, "host-doctor.json")
	if err := os.WriteFile(result.HostDoctorPath, []byte(hostDoctorOutput), 0o644); err != nil {
		return result, err
	}
	log.Log("INFO", "chat", "host-doctor", "ok", result.HostDoctorPath)

	transcriptPath := filepath.Join(artifactRoot, "transcript.md")
	result.TranscriptPath = transcriptPath
	chatEnv := map[string]string{
		"ANTHROPIC_API_KEY": os.Getenv("ANTHROPIC_API_KEY"),
	}
	for index, prompt := range opts.Prompts {
		fullPrompt := renderChatPrompt(result.Turns, prompt)
		response, err := dockerExec(containerName, "/workspace/project", chatEnv, []string{
			"claude", "-p", fullPrompt, "--dangerously-skip-permissions", "--output-format", "text",
		})
		if err != nil {
			log.Log("ERROR", "chat", fmt.Sprintf("turn-%02d", index+1), "fail", err.Error())
			return result, err
		}
		result.Turns = append(result.Turns, ChatTurn{
			Prompt:   prompt,
			Response: response,
		})
		if err := writeChatTranscript(transcriptPath, result, installOutput, hostDoctorOutput); err != nil {
			return result, err
		}
		log.Log("INFO", "chat", fmt.Sprintf("turn-%02d", index+1), "ok", preview(response))
	}

	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(artifactRoot, "chat.json"), raw, 0o644); err != nil {
		return result, err
	}
	log.Log("INFO", "chat", "complete", "ok", transcriptPath)
	return result, nil
}

func defaultChatPrompts() []string {
	return []string{
		"Inspect this workspace and explain how you perceive the `epics` CLI and the installed Epic. Reference specific files or commands you inspected.",
		"Based on what you saw, what concrete improvements or feature requests would you suggest for the `epics` tool, host integration, or installed Epic UX?",
		"List any bugs, confusing behavior, or rough edges you noticed. Separate confirmed issues from speculative risks.",
	}
}

func renderChatPrompt(turns []ChatTurn, nextPrompt string) string {
	var b strings.Builder
	b.WriteString("You are participating in a live-chat evaluation of the `epics` CLI inside a Docker workspace.\n")
	b.WriteString("Inspect the repository and installed Epic directly when useful. Keep answers concrete and concise.\n")
	if len(turns) > 0 {
		b.WriteString("\nConversation so far:\n")
		for _, turn := range turns {
			b.WriteString("\nUser: ")
			b.WriteString(turn.Prompt)
			b.WriteString("\nClaude: ")
			b.WriteString(strings.TrimSpace(turn.Response))
			b.WriteString("\n")
		}
	}
	b.WriteString("\nUser: ")
	b.WriteString(nextPrompt)
	return b.String()
}

func writeChatTranscript(path string, result ChatResult, installOutput, hostDoctorOutput string) error {
	var b strings.Builder
	b.WriteString("# Live Chat Transcript\n\n")
	b.WriteString("- Run ID: `" + result.RunID + "`\n")
	b.WriteString("- Container: `" + result.ContainerName + "`\n")
	b.WriteString("- Image: `" + result.ImageTag + "`\n")
	b.WriteString("- Workspace: `" + filepath.ToSlash(result.WorkspaceDir) + "`\n")
	if result.ShellCommand != "" {
		b.WriteString("- Shell: `" + result.ShellCommand + "`\n")
	}
	if result.CleanupCommand != "" {
		b.WriteString("- Cleanup: `" + result.CleanupCommand + "`\n")
	} else {
		b.WriteString("- Cleanup: container removed automatically after chat\n")
	}
	b.WriteString("\n")
	b.WriteString("## Install Output\n\n```text\n")
	b.WriteString(strings.TrimSpace(installOutput))
	b.WriteString("\n```\n\n")
	b.WriteString("## Host Doctor\n\n```json\n")
	b.WriteString(strings.TrimSpace(hostDoctorOutput))
	b.WriteString("\n```\n")
	for index, turn := range result.Turns {
		b.WriteString(fmt.Sprintf("\n## Turn %d\n\n", index+1))
		b.WriteString("**User**\n\n")
		b.WriteString(turn.Prompt)
		b.WriteString("\n\n**Claude**\n\n```text\n")
		b.WriteString(strings.TrimSpace(turn.Response))
		b.WriteString("\n```\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func startChatContainer(containerName, imageTag, workspaceDir string) error {
	args := []string{
		"run", "-d", "--rm",
		"--name", containerName,
		"-v", workspaceDir + ":/workspace",
		"-w", "/workspace/project",
		"-e", "HOME=/workspace/.claude-home",
		"-e", "XDG_CONFIG_HOME=/workspace/.claude-home/.config",
	}
	if uid, gid, ok := currentUserIDs(); ok {
		args = append(args, "--user", uid+":"+gid)
	}
	args = append(args, imageTag, "sleep", "infinity")
	output, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func removeContainer(containerName string) {
	_ = exec.Command("docker", "rm", "-f", containerName).Run()
}

func dockerExec(containerName, workdir string, env map[string]string, command []string) (string, error) {
	args := []string{"exec"}
	if workdir != "" {
		args = append(args, "-w", workdir)
	}
	for _, key := range sortedEnvKeys(env) {
		args = append(args, "-e", key+"="+env[key])
	}
	args = append(args, containerName)
	args = append(args, command...)

	output, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker exec failed: %s", strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
