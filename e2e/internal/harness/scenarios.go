package harness

func DefaultScenarios() []Scenario {
	return []Scenario{
		{
			Name:         "claude-hello-world",
			Description:  "Verify that Claude Code can run a headless print-mode prompt in the Claude container.",
			Tags:         []string{"claude", "live"},
			ImageProfile: "claude",
			RequiredEnv:  []string{"ANTHROPIC_API_KEY"},
			Steps: []Step{
				{
					Name:           "claude-hello",
					Program:        "claude",
					Args:           []string{"-p", "Respond exactly 'Hello world!' and nothing else.", "--dangerously-skip-permissions", "--output-format", "text"},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutEquals:   "Hello world!",
				},
			},
		},
		{
			Name:         "claude-install-remote-epic",
			Description:  "Install a real Epic from the public GitHub repo into a small Claude workspace project.",
			Tags:         []string{"claude", "install", "remote", "live"},
			ImageProfile: "claude",
			RequiredEnv:  []string{"ANTHROPIC_API_KEY"},
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
			},
			Steps: []Step{
				{
					Name:           "install-remote",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "https://github.com/agentepics/epics/tree/main/autonomous-coding"},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Autonomous Coding for claude into .claude/skills/autonomous-coding"},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.claude/skills/autonomous-coding/SKILL.md", MustExist: true},
				{Path: "project/.claude/skills/autonomous-coding/EPIC.md", MustExist: true},
				{Path: "project/.claude/skills/autonomous-coding/state.json", MustExist: true, Contains: []string{"current_plan", "001-initial.md"}},
				{Path: "project/.claude/skills/autonomous-coding/plans/001-initial.md", MustExist: true},
				{Path: "project/.claude/skills/autonomous-coding/log", MustExist: true},
				{Path: "project/.claude/skills/autonomous-coding/runtime/install.json", MustExist: true, Contains: []string{"\"trigger\": \"install\"", "\"epicId\": \"autonomous-coding\"", "hooks/install.md"}},
				{Path: "project/.claude/commands/epics-resume.md", MustExist: true},
				{Path: "project/.epics/installs.json", MustExist: true, Contains: []string{"autonomous-coding", "\"host\": \"claude\"", ".claude/skills/autonomous-coding"}},
			},
		},
		{
			Name:         "claude-install-hook-fires",
			Description:  "Validate that the EPIC-spec install trigger runs during installation.",
			Tags:         []string{"claude", "hooks"},
			ImageProfile: "claude",
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
				{From: "examples/fixtures/install-hook-epic", To: "fixtures/install-hook-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-hook-epic",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "../fixtures/install-hook-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Install Hook Epic for claude into .claude/skills/install-hook-epic"},
				},
				{
					Name:                     "info-installed-hook-epic",
					Workdir:                  "project",
					Args:                     []string{"info", "install-hook-epic"},
					ExpectNoWorkspaceChanges: true,
					ExpectExitCode:           0,
					StdoutContains:           []string{"Host: claude", "Installed: .claude/skills/install-hook-epic"},
				},
				{
					Name:                     "resume-installed-hook-epic",
					Workdir:                  "project",
					Args:                     []string{"resume", "install-hook-epic"},
					ExpectNoWorkspaceChanges: true,
					ExpectExitCode:           0,
					StdoutContains:           []string{"Epic: Install Hook Epic", "Resume hint: review EPIC.md and SKILL.md to re-enter the workflow."},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.claude/skills/install-hook-epic/runtime/install.json", MustExist: true, Contains: []string{"\"trigger\": \"install\"", "\"epicId\": \"install-hook-epic\""}},
				{Path: "project/.claude/skills/install-hook-epic/runtime/install-hook-output.json", MustExist: true, Contains: []string{"\"trigger\":\"install\"", "\"epic_id\":\"install-hook-epic\""}},
			},
		},
		{
			Name:         "claude-install-hook-failure-rolls-back",
			Description:  "Validate that a failing install hook aborts installation without leaving partial installed state behind.",
			Tags:         []string{"claude", "hooks", "negative"},
			ImageProfile: "claude",
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
				{From: "examples/fixtures/failing-install-hook-epic", To: "fixtures/failing-install-hook-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-failing-hook-epic",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "../fixtures/failing-install-hook-epic"},
					ExpectExitCode: 1,
					StderrContains: []string{"intentional install hook failure"},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.claude/skills/failing-install-hook-epic", MustExist: false},
				{Path: "project/.epics/installs.json", MustExist: false},
			},
		},
		{
			Name:         "claude-prompt-install-hook-fires",
			Description:  "Validate that a prompt-based EPIC install hook runs through real Claude Code during installation.",
			Tags:         []string{"claude", "hooks", "live"},
			ImageProfile: "claude",
			RequiredEnv:  []string{"ANTHROPIC_API_KEY"},
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
				{From: "examples/fixtures/prompt-install-hook-epic", To: "fixtures/prompt-install-hook-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-prompt-hook-epic",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "../fixtures/prompt-install-hook-epic"},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Prompt Install Hook Epic for claude into .claude/skills/prompt-install-hook-epic"},
				},
				{
					Name:                     "info-installed-prompt-hook-epic",
					Workdir:                  "project",
					Args:                     []string{"info", "prompt-install-hook-epic"},
					ExpectNoWorkspaceChanges: true,
					ExpectExitCode:           0,
					StdoutContains:           []string{"Host: claude", "Installed: .claude/skills/prompt-install-hook-epic"},
				},
				{
					Name:                     "resume-installed-prompt-hook-epic",
					Workdir:                  "project",
					Args:                     []string{"resume", "prompt-install-hook-epic"},
					ExpectNoWorkspaceChanges: true,
					ExpectExitCode:           0,
					StdoutContains:           []string{"Epic: Prompt Install Hook Epic", "Resume hint: review EPIC.md and SKILL.md to re-enter the workflow."},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.claude/skills/prompt-install-hook-epic/runtime/install.json", MustExist: true, Contains: []string{"\"trigger\": \"install\"", "\"epicId\": \"prompt-install-hook-epic\""}},
				{Path: "project/.claude/skills/prompt-install-hook-epic/runtime/prompt-hook-output.json", MustExist: true, Contains: []string{"\"status\":\"ok\"", "\"trigger\":\"install\"", "\"epic_id\":\"prompt-install-hook-epic\""}},
			},
		},
		{
			Name:         "claude-can-read-installed-epic",
			Description:  "Validate that Claude can discover and read the installed Epic from the project workspace.",
			Tags:         []string{"claude", "live", "read"},
			ImageProfile: "claude",
			RequiredEnv:  []string{"ANTHROPIC_API_KEY"},
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
			},
			Steps: []Step{
				{
					Name:           "install-remote",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "https://github.com/agentepics/epics/tree/main/autonomous-coding"},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
				},
				{
					Name:    "claude-read-epic",
					Program: "claude",
					Workdir: "project",
					Args: []string{
						"-p",
						"Inspect the current workspace. Look under .claude/skills and return compact JSON only in the shape {\"epics\":[{\"path\":\"...\",\"title\":\"...\",\"slug\":\"...\"}]}.",
						"--dangerously-skip-permissions",
						"--output-format",
						"text",
					},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{"autonomous-coding", "Autonomous Coding", ".claude/skills/autonomous-coding"},
				},
			},
		},
		{
			Name:         "claude-can-use-epics-helpers",
			Description:  "Validate that Claude can use the new epics helper commands against an installed Epic workspace.",
			Tags:         []string{"claude", "live", "helpers"},
			ImageProfile: "claude",
			RequiredEnv:  []string{"ANTHROPIC_API_KEY"},
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-local",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "../fixtures/resume-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Resume Epic for claude into .claude/skills/resume-epic"},
				},
				{
					Name:           "state-get-next-step",
					Workdir:        "project/.claude/skills/resume-epic",
					Args:           []string{"state", "get", "nextStep"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Verify the generated summary output"},
				},
				{
					Name:           "plan-current",
					Workdir:        "project/.claude/skills/resume-epic",
					Args:           []string{"plan", "current"},
					ExpectExitCode: 0,
					StdoutContains: []string{"# Current Plan", "Review recent logs before continuing"},
				},
				{
					Name:           "host-doctor",
					Workdir:        "project",
					Args:           []string{"--json", "host", "doctor", "claude"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"claude-managed-dir"`, `"claude-commands"`, `"claude-instructions"`, `"claude-skills"`, `"status": "ok"`},
				},
				{
					Name:    "claude-inspect-helper-files",
					Program: "claude",
					Workdir: "project",
					Args: []string{
						"-p",
						"Inspect this workspace. Read .claude/commands and .claude/skills/resume-epic, then return compact JSON only with keys commands and skill_root. Include the exact paths for epics-resume and epics-doctor if they exist.",
						"--dangerously-skip-permissions",
						"--output-format",
						"text",
					},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{".claude/commands/epics-resume.md", ".claude/commands/epics-doctor.md", ".claude/skills/resume-epic"},
				},
			},
		},
		{
			Name:         "init-empty-workspace",
			Description:  "Initialize an empty workspace into a minimal Epic package.",
			Tags:         []string{"cli", "core"},
			ImageProfile: "cli",
			Steps: []Step{
				{
					Name:           "init",
					Args:           []string{"init"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Initialized Epic package:"},
				},
			},
			Files: []FileAssertion{
				{Path: "SKILL.md", MustExist: true},
				{Path: "EPIC.md", MustExist: true},
			},
		},
		{
			Name:         "validate-valid-fixture",
			Description:  "Validate the known-good Epic fixture.",
			Tags:         []string{"cli", "core"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/valid-epic", To: "fixtures/valid-epic"},
			},
			Steps: []Step{
				{
					Name:           "validate",
					Args:           []string{"validate", "./fixtures/valid-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Valid Epic is valid."},
				},
			},
		},
		{
			Name:         "validate-invalid-fixture",
			Description:  "Validate the invalid fixture and assert a failure exit code.",
			Tags:         []string{"cli", "core"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/invalid-missing-epic", To: "fixtures/invalid-missing-epic"},
			},
			Steps: []Step{
				{
					Name:           "validate-invalid",
					Args:           []string{"validate", "./fixtures/invalid-missing-epic"},
					ExpectExitCode: 1,
					StdoutContains: []string{"missing required file EPIC.md"},
				},
			},
		},
		{
			Name:         "install-local-fixture",
			Description:  "Install a local Epic fixture into Claude's local skills folder.",
			Tags:         []string{"cli", "install", "claude"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-local",
					Args:           []string{"install", "--host", "claude", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Resume Epic for claude into .claude/skills/resume-epic", "Claude workspace setup complete."},
				},
			},
			Files: []FileAssertion{
				{Path: ".claude/skills/resume-epic/SKILL.md", MustExist: true},
				{Path: ".claude/skills/resume-epic/EPIC.md", MustExist: true},
				{Path: ".claude/commands/epics-resume.md", MustExist: true},
				{Path: ".epics/installs.json", MustExist: true, Contains: []string{"resume-epic", "\"host\": \"claude\"", ".claude/skills/resume-epic"}},
			},
		},
		{
			Name:         "gemini-install-local-epic",
			Description:  "Install a local Epic fixture into Gemini's local skills folder.",
			Tags:         []string{"cli", "install", "gemini"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-local",
					Args:           []string{"install", "--host", "gemini", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Resume Epic for gemini into .gemini/skills/resume-epic", "Gemini workspace setup complete."},
				},
			},
			Files: []FileAssertion{
				{Path: ".gemini/skills/resume-epic/SKILL.md", MustExist: true},
				{Path: ".gemini/skills/resume-epic/EPIC.md", MustExist: true},
				{Path: ".gemini/commands/epics-resume.md", MustExist: true},
				{Path: ".gemini/commands/epics-info.md", MustExist: true},
				{Path: ".gemini/commands/epics-doctor.md", MustExist: true},
				{Path: "GEMINI.md", MustExist: true, Contains: []string{"Epics CLI Guidance"}},
				{Path: ".epics/installs.json", MustExist: true, Contains: []string{"resume-epic", "\"host\": \"gemini\"", ".gemini/skills/resume-epic"}},
			},
		},
		{
			Name:         "opencode-install-local-epic",
			Description:  "Install a local Epic fixture into OpenCode's local skills folder.",
			Tags:         []string{"cli", "install", "opencode"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-local",
					Args:           []string{"install", "--host", "opencode", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Resume Epic for opencode into .opencode/skills/resume-epic", "Opencode workspace setup complete."},
				},
			},
			Files: []FileAssertion{
				{Path: ".opencode/skills/resume-epic/SKILL.md", MustExist: true},
				{Path: ".opencode/skills/resume-epic/EPIC.md", MustExist: true},
				{Path: ".opencode/commands/epics-resume.md", MustExist: true},
				{Path: ".opencode/commands/epics-info.md", MustExist: true},
				{Path: ".opencode/commands/epics-doctor.md", MustExist: true},
				{Path: "AGENTS.md", MustExist: true, Contains: []string{"Epics CLI Guidance"}},
				{Path: ".epics/installs.json", MustExist: true, Contains: []string{"resume-epic", "\"host\": \"opencode\"", ".opencode/skills/resume-epic"}},
			},
		},
		{
			Name:         "install-registry-source",
			Description:  "Install a registry-backed Epic by repo-style source path into Claude's local skills folder.",
			Tags:         []string{"cli", "install", "registry", "claude"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "registry", To: "registry"},
			},
			Steps: []Step{
				{
					Name:           "install-registry",
					Args:           []string{"install", "--host", "claude", "github.com/agentepics/epics/autonomous-coding"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Autonomous Coding for claude into .claude/skills/autonomous-coding"},
				},
			},
			Files: []FileAssertion{
				{Path: ".claude/skills/autonomous-coding/SKILL.md", MustExist: true},
				{Path: ".claude/skills/autonomous-coding/EPIC.md", MustExist: true},
				{Path: ".epics/installs.json", MustExist: true, Contains: []string{"autonomous-coding", "\"host\": \"claude\"", ".claude/skills/autonomous-coding"}},
			},
		},
		{
			Name:         "info-installed-epic",
			Description:  "Read Claude-installed registry metadata back through the info command.",
			Tags:         []string{"cli", "info"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "registry", To: "registry"},
			},
			Steps: []Step{
				{
					Name:           "install-registry",
					Args:           []string{"install", "--host", "claude", "github.com/agentepics/epics/autonomous-coding"},
					ExpectExitCode: 0,
				},
				{
					Name:           "info",
					Args:           []string{"info", "autonomous-coding"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Title: Autonomous Coding", "Slug: autonomous-coding", "Source: github.com/agentepics/epics/autonomous-coding", "Version:", "Digest:", "Host: claude", "Installed: .claude/skills/autonomous-coding"},
				},
			},
		},
		{
			Name:         "install-interactive-host-selection",
			Description:  "Prompt for the target host when install runs interactively without --host.",
			Tags:         []string{"cli", "install", "interactive", "claude"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-interactive",
					Args:           []string{"install", "./fixtures/resume-epic"},
					Stdin:          "claude\n",
					Env:            map[string]string{"EPICS_FORCE_INTERACTIVE": "1"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Select host:", "Host [claude]:", "Installed Resume Epic for claude into .claude/skills/resume-epic"},
				},
			},
			Files: []FileAssertion{
				{Path: ".claude/skills/resume-epic/EPIC.md", MustExist: true},
				{Path: ".epics/installs.json", MustExist: true, Contains: []string{"\"host\": \"claude\""}},
			},
		},
		{
			Name:         "install-missing-host-noninteractive",
			Description:  "Fail install instead of guessing the host when stdin is non-interactive.",
			Tags:         []string{"cli", "install", "negative"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:              "install-without-host",
					Args:              []string{"install", "./fixtures/resume-epic"},
					ExpectExitCode:    1,
					StderrContains:    []string{"install requires --host <host> when stdin is not interactive"},
					StdoutNotContains: []string{"Installed Resume Epic"},
				},
			},
		},
		{
			Name:         "resume-stateful-epic",
			Description:  "Resume from a fixture with state, plans, and logs.",
			Tags:         []string{"cli", "resume"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "resume",
					Args:           []string{"resume", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Next step: Verify the generated summary output", "Current plan: plans/001-current.md"},
				},
			},
		},
		{
			Name:         "status-stateful-epic",
			Description:  "Show a compact status summary for a stateful Epic fixture.",
			Tags:         []string{"cli", "status"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "status",
					Args:           []string{"status", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Epic: Resume Epic", "Current plan: plans/001-current.md", "Next step: Verify the generated summary output", "Latest log: log/2026-03-08-01.md"},
				},
			},
		},
		{
			Name:         "doctor-empty-workspace",
			Description:  "Run doctor in an empty workspace and assert generic checks only.",
			Tags:         []string{"cli", "doctor"},
			ImageProfile: "cli",
			Steps: []Step{
				{
					Name:              "doctor",
					Args:              []string{"--json", "doctor"},
					ExpectExitCode:    0,
					StdoutContains:    []string{`"managed-dir"`, `"workspace-write"`, `"authored-package"`, `"installed-epics"`},
					StdoutNotContains: []string{"claude"},
				},
			},
		},
		{
			Name:         "host-doctor-claude-installed-epic",
			Description:  "Run host doctor against a Claude workspace after installing an Epic.",
			Tags:         []string{"cli", "doctor", "host", "claude"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-local",
					Args:           []string{"install", "--host", "claude", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
				},
				{
					Name:                     "host-doctor",
					Args:                     []string{"--json", "host", "doctor", "claude"},
					ExpectNoWorkspaceChanges: true,
					ExpectExitCode:           0,
					StdoutContains:           []string{`"claude-managed-dir"`, `"claude-commands"`, `"claude-instructions"`, `"claude-skills"`, `"status": "ok"`},
				},
			},
		},
		{
			Name:         "doctor-warns-on-missing-local-source",
			Description:  "Warn when workspace install metadata points to a missing local Epic source.",
			Tags:         []string{"cli", "doctor", "warning"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "install-local",
					Args:           []string{"install", "--host", "claude", "./fixtures/resume-epic"},
					ExpectExitCode: 0,
				},
				{
					Name:           "remove-local-source-and-doctor",
					Program:        "sh",
					Args:           []string{"-lc", "rm -rf ./fixtures/resume-epic && epics doctor"},
					ExpectExitCode: 0,
					StdoutContains: []string{"WARNING: install-sources - missing sources: resume-epic@claude ->", "INSTALLED-EPICS: installed-epics - workspace metadata tracks 1 installed Epic(s): resume-epic@claude"},
				},
			},
		},
		{
			Name:         "state-helpers-roundtrip",
			Description:  "Read and write Epic state using state/core.json precedence.",
			Tags:         []string{"cli", "state"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "state-get-next-step",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"state", "get", "nextStep"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Verify the generated summary output"},
				},
				{
					Name:           "state-set-phase",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"state", "set", "phase.current", "\"planning\""},
					ExpectExitCode: 0,
					StdoutContains: []string{"state/core.json"},
				},
				{
					Name:           "state-get-phase-json",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"--json", "state", "get", "phase.current"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"key": "phase.current"`, `"planning"`},
				},
			},
			Files: []FileAssertion{
				{Path: "fixtures/resume-epic/state/core.json", MustExist: true, Contains: []string{`"phase"`, `"current": "planning"`, `"nextStep": "Verify the generated summary output"`}},
			},
		},
		{
			Name:         "plan-helpers-current-create-list",
			Description:  "Resolve the current plan, create a new plan, and list plans.",
			Tags:         []string{"cli", "plan"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "plan-current",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"plan", "current"},
					ExpectExitCode: 0,
					StdoutContains: []string{"# Current Plan", "Verify the generated summary output"},
				},
				{
					Name:           "plan-create",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"plan", "create", "Follow-up", "plan"},
					ExpectExitCode: 0,
					StdoutContains: []string{"plans/002-follow-up-plan.md"},
				},
				{
					Name:           "plan-list-json",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"--json", "plan", "list"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"path": "plans/001-current.md"`, `"path": "plans/002-follow-up-plan.md"`, `"title": "Follow-up plan"`},
				},
			},
			Files: []FileAssertion{
				{Path: "fixtures/resume-epic/plans/002-follow-up-plan.md", MustExist: true, Contains: []string{"# Follow-up plan"}},
			},
		},
		{
			Name:         "log-helpers-create-recent",
			Description:  "Create a new log entry and read it back through log recent.",
			Tags:         []string{"cli", "log"},
			ImageProfile: "cli",
			Copies: []CopySpec{
				{From: "examples/fixtures/resume-epic", To: "fixtures/resume-epic"},
			},
			Steps: []Step{
				{
					Name:           "log-create",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"log", "create", "Session", "1"},
					ExpectExitCode: 0,
					StdoutContains: []string{"log/"},
				},
				{
					Name:           "log-recent-json",
					Workdir:        "fixtures/resume-epic",
					Args:           []string{"--json", "log", "recent", "1"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"path": "log/`, `title: Session 1`},
				},
			},
		},
		{
			Name:         "cron-validate-fixture",
			Description:  "Validate cron.d entries in an Epic fixture workspace.",
			Tags:         []string{"cli", "cron"},
			ImageProfile: "cli",
			SeedFiles: map[string]string{
				"cron.d/nightly": "*/15 9 * * 1-5 scripts/run.sh\n",
				"scripts/run.sh": "#!/bin/sh\necho ok\n",
			},
			Steps: []Step{
				{
					Name:           "cron-validate",
					Args:           []string{"cron", "validate"},
					ExpectExitCode: 0,
					StdoutContains: []string{"cron.d is valid."},
				},
				{
					Name:           "cron-validate-json",
					Args:           []string{"--json", "cron", "validate"},
					ExpectExitCode: 0,
					StdoutContains: []string{"null"},
				},
			},
		},
		{
			Name:         "host-setup-claude-additive",
			Description:  "Generate Claude setup output in a clean workspace.",
			Tags:         []string{"cli", "host", "claude"},
			ImageProfile: "cli",
			Steps: []Step{
				{
					Name:           "host-setup",
					Args:           []string{"host", "setup", "claude"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Claude workspace setup complete."},
				},
			},
			Files: []FileAssertion{
				{Path: ".claude/commands/epics-resume.md", MustExist: true},
				{Path: ".epics/hosts/claude/README.md", MustExist: true},
				{Path: "CLAUDE.md", MustExist: true, Contains: []string{"`epics` CLI"}},
			},
		},
		{
			Name:         "host-setup-claude-preserve-existing",
			Description:  "Append Epic guidance to an existing CLAUDE.md during Claude setup.",
			Tags:         []string{"cli", "host", "claude"},
			ImageProfile: "cli",
			SeedFiles: map[string]string{
				"CLAUDE.md": "# Existing\n",
			},
			Steps: []Step{
				{
					Name:           "host-setup",
					Args:           []string{"host", "setup", "claude"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Claude workspace setup complete.", "CLAUDE.md"},
				},
			},
			Files: []FileAssertion{
				{Path: "CLAUDE.md", MustExist: true, Contains: []string{"# Existing", "Epics CLI Guidance"}},
				{Path: ".claude/commands/epics-info.md", MustExist: true},
				{Path: ".epics/hosts/claude/README.md", MustExist: true, Contains: []string{".claude/skills/<slug>/"}},
			},
		},
	}
}
