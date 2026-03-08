package harness

func DefaultScenarios() []Scenario {
	return []Scenario{
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
			Name:         "doctor-empty-workspace",
			Description:  "Run doctor in an empty workspace and assert generic checks only.",
			Tags:         []string{"cli", "doctor"},
			ImageProfile: "cli",
			Steps: []Step{
				{
					Name:              "doctor",
					Args:              []string{"--json", "doctor"},
					ExpectExitCode:    0,
					StdoutContains:    []string{`"managed-dir"`, `"workspace-write"`},
					StdoutNotContains: []string{"claude"},
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
			Description:  "Preserve an existing CLAUDE.md during Claude setup.",
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
					StdoutContains: []string{"Preserved existing:", "CLAUDE.md"},
				},
			},
			Files: []FileAssertion{
				{Path: "CLAUDE.md", MustExist: true, Equals: "# Existing\n"},
				{Path: ".claude/commands/epics-info.md", MustExist: true},
				{Path: ".epics/hosts/claude/README.md", MustExist: true, Contains: []string{".claude/skills/<slug>/"}},
			},
		},
	}
}
