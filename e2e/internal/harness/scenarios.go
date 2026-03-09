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
			Name:         "claude-epicsd-webhook-dispatch",
			Description:  "Start epicsd in the live Claude container, register the workspace, deliver a localhost webhook, and verify a successful daemon dispatch through Claude.",
			Tags:         []string{"claude", "live", "daemon"},
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
					Name:    "start-epicsd",
					Program: "sh",
					Workdir: "project",
					Args:    []string{"-lc", `mkdir -p "$EPICSD_HOME" .epicsd-artifacts; epicsd > "$EPICSD_HOME/epicsd.log" 2>&1 & pid=$!; trap 'kill "$pid" >/dev/null 2>&1 || true' EXIT; echo "$pid" > .epicsd-artifacts/epicsd.pid; for i in $(seq 1 50); do [ -S "$EPICSD_HOME/epicsd.sock" ] && break; sleep 0.2; done; [ -S "$EPICSD_HOME/epicsd.sock" ] || { cat "$EPICSD_HOME/epicsd.log"; exit 1; }; cp "$EPICSD_HOME/config.json" .epicsd-artifacts/config.json; epics --json daemon status > .epicsd-artifacts/status.json; WS=$(epics --json workspace register . --name live-project | tee .epicsd-artifacts/workspace.json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])'); cp "$EPICSD_HOME/workspaces.json" .epicsd-artifacts/workspaces.json; epics --json route upsert --type webhook --workspace "$WS" --epic resume-epic --provider github --endpoint live-project --preferred-adapter claude --auth bearer --secret live-token > .epicsd-artifacts/route.json; cp "$EPICSD_HOME/routes.json" .epicsd-artifacts/routes.json; ADDR=$(python3 -c 'import json;print(json.load(open(".epicsd-artifacts/status.json"))["webhookHTTPAddr"])'); python3 -c 'import sys,urllib.request; addr=sys.argv[1]; req=urllib.request.Request(f"http://{addr}/v1/webhooks/github/live-project", data=b"{\"event\":\"ping\"}", method="POST", headers={"Authorization":"Bearer live-token","Content-Type":"application/json","X-GitHub-Delivery":"live-delivery-1"}); resp=urllib.request.urlopen(req); print(resp.status); print(resp.read().decode())' "$ADDR" > .epicsd-artifacts/webhook.txt; for i in $(seq 1 60); do epics --json run list --limit 5 > .epicsd-artifacts/runs.json; cp "$EPICSD_HOME/epicsd.log" .epicsd-artifacts/epicsd.log; python3 -c 'import json,sys; runs=json.load(open(".epicsd-artifacts/runs.json")) or []; sys.exit(0 if any(r.get("routeId")=="webhook:github:live-project" and r.get("outcome")=="succeeded" and r.get("adapter")=="claude" for r in runs) else 1)' && break; sleep 1; done; python3 -c 'import json,sys; runs=json.load(open(".epicsd-artifacts/runs.json")) or []; sys.exit(0 if any(r.get("routeId")=="webhook:github:live-project" and r.get("outcome")=="succeeded" and r.get("adapter")=="claude" for r in runs) else 1)' || { cat .epicsd-artifacts/runs.json; cat .epicsd-artifacts/epicsd.log; exit 1; }; cat .epicsd-artifacts/status.json; cat .epicsd-artifacts/workspace.json; cat .epicsd-artifacts/route.json; cat .epicsd-artifacts/webhook.txt; cat .epicsd-artifacts/runs.json`},
					Env: map[string]string{
						"EPICSD_HOME": "/tmp/epicsd-home",
					},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"status": "ok"`, `"adminSocketPath": "/tmp/epicsd-home/epicsd.sock"`, `"id": "ws_`, `"displayName": "live-project"`, `"id": "webhook:github:live-project"`, `"selectedAdapter": "claude"`, "202", `"queued":true`, `"routeId": "webhook:github:live-project"`, `"outcome": "succeeded"`, `"adapter": "claude"`},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.epicsd-artifacts/config.json", MustExist: true, Contains: []string{"127.0.0.1", "epicsd.sock"}},
				{Path: "project/.epicsd-artifacts/workspaces.json", MustExist: true, Contains: []string{"live-project", "/workspace/project"}},
				{Path: "project/.epicsd-artifacts/routes.json", MustExist: true, Contains: []string{"webhook:github:live-project", "\"epicSlug\": \"resume-epic\""}},
				{Path: "project/.epicsd-artifacts/status.json", MustExist: true, Contains: []string{`"status": "ok"`, `"webhookHTTPAddr": "127.0.0.1:`}},
				{Path: "project/.epicsd-artifacts/runs.json", MustExist: true, Contains: []string{`"routeId": "webhook:github:live-project"`, `"outcome": "succeeded"`}},
				{Path: "project/.epicsd-artifacts/epicsd.log", MustExist: true, Contains: []string{"route=webhook:github:live-project", "adapter=claude"}},
			},
		},
		{
			Name:         "claude-epicsd-restart-recovery",
			Description:  "Run a live Claude webhook route through epicsd, restart the daemon, and prove persisted workspace/route state survives into a second successful dispatch.",
			Tags:         []string{"claude", "live", "daemon", "restart"},
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
					Name:    "restart-recovery",
					Program: "sh",
					Workdir: "project",
					Args: []string{"-lc", `
mkdir -p "$EPICSD_HOME" .epicsd-restart
pids=""
start_daemon() {
  epicsd >> "$EPICSD_HOME/epicsd.log" 2>&1 &
  current_pid=$!
  pids="$pids $current_pid"
  for i in $(seq 1 50); do
    [ -S "$EPICSD_HOME/epicsd.sock" ] && return 0
    sleep 0.2
  done
  cat "$EPICSD_HOME/epicsd.log"
  return 1
}
stop_daemon() {
  kill "$current_pid" >/dev/null 2>&1 || true
  wait "$current_pid" >/dev/null 2>&1 || true
  for i in $(seq 1 50); do
    [ ! -S "$EPICSD_HOME/epicsd.sock" ] && return 0
    sleep 0.2
  done
  return 1
}
trap 'for p in $pids; do kill "$p" >/dev/null 2>&1 || true; done' EXIT

start_daemon
epics --json daemon status > .epicsd-restart/status-before.json
WS=$(epics --json workspace register . --name restart-project | tee .epicsd-restart/workspace-before.json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
epics --json route upsert --type webhook --workspace "$WS" --epic resume-epic --provider github --endpoint restart-project --preferred-adapter claude --auth bearer --secret restart-token > .epicsd-restart/route-before.json
cp "$EPICSD_HOME/workspaces.json" .epicsd-restart/workspaces-before.json
cp "$EPICSD_HOME/routes.json" .epicsd-restart/routes-before.json
ADDR=$(python3 -c 'import json;print(json.load(open(".epicsd-restart/status-before.json"))["webhookHTTPAddr"])')
python3 -c 'import sys,urllib.request; addr=sys.argv[1]; req=urllib.request.Request(f"http://{addr}/v1/webhooks/github/restart-project", data=b"{\"event\":\"before-restart\"}", method="POST", headers={"Authorization":"Bearer restart-token","Content-Type":"application/json","X-GitHub-Delivery":"restart-delivery-1"}); resp=urllib.request.urlopen(req); print(resp.status); print(resp.read().decode())' "$ADDR" > .epicsd-restart/webhook-before.txt
for i in $(seq 1 60); do
  epics --json run list --limit 10 > .epicsd-restart/runs-before.json
  python3 -c 'import json,sys; runs=json.load(open(".epicsd-restart/runs-before.json")) or []; hits=[r for r in runs if r.get("routeId")=="webhook:github:restart-project" and r.get("outcome")=="succeeded" and r.get("adapter")=="claude"]; sys.exit(0 if len(hits) >= 1 else 1)' && break
  sleep 1
done
python3 -c 'import json,sys; runs=json.load(open(".epicsd-restart/runs-before.json")) or []; hits=[r for r in runs if r.get("routeId")=="webhook:github:restart-project" and r.get("outcome")=="succeeded" and r.get("adapter")=="claude"]; sys.exit(0 if len(hits) >= 1 else 1)'

stop_daemon
start_daemon
epics --json daemon status > .epicsd-restart/status-after.json
epics --json workspace inspect "$WS" > .epicsd-restart/workspace-after.json
epics --json route inspect webhook:github:restart-project > .epicsd-restart/route-after.json
cp "$EPICSD_HOME/workspaces.json" .epicsd-restart/workspaces-after.json
cp "$EPICSD_HOME/routes.json" .epicsd-restart/routes-after.json
ADDR=$(python3 -c 'import json;print(json.load(open(".epicsd-restart/status-after.json"))["webhookHTTPAddr"])')
python3 -c 'import sys,urllib.request; addr=sys.argv[1]; req=urllib.request.Request(f"http://{addr}/v1/webhooks/github/restart-project", data=b"{\"event\":\"after-restart\"}", method="POST", headers={"Authorization":"Bearer restart-token","Content-Type":"application/json","X-GitHub-Delivery":"restart-delivery-2"}); resp=urllib.request.urlopen(req); print(resp.status); print(resp.read().decode())' "$ADDR" > .epicsd-restart/webhook-after.txt
for i in $(seq 1 60); do
  epics --json run list --limit 20 > .epicsd-restart/runs-after.json
  cp "$EPICSD_HOME/epicsd.log" .epicsd-restart/epicsd.log
  python3 -c 'import json,sys; runs=json.load(open(".epicsd-restart/runs-after.json")) or []; hits=[r for r in runs if r.get("routeId")=="webhook:github:restart-project" and r.get("outcome")=="succeeded" and r.get("adapter")=="claude"]; sys.exit(0 if len(hits) >= 2 else 1)' && break
  sleep 1
done
python3 -c 'import json,sys; runs=json.load(open(".epicsd-restart/runs-after.json")) or []; hits=[r for r in runs if r.get("routeId")=="webhook:github:restart-project" and r.get("outcome")=="succeeded" and r.get("adapter")=="claude"]; summary={"successes": len(hits), "workspaceId": hits[0]["workspaceId"] if hits else "", "routeId": "webhook:github:restart-project"}; open(".epicsd-restart/recovery-summary.json","w").write(json.dumps(summary, indent=2) + "\n"); sys.exit(0 if len(hits) >= 2 else 1)'
cat .epicsd-restart/status-before.json
cat .epicsd-restart/status-after.json
cat .epicsd-restart/workspace-before.json
cat .epicsd-restart/workspace-after.json
cat .epicsd-restart/route-before.json
cat .epicsd-restart/route-after.json
cat .epicsd-restart/webhook-before.txt
cat .epicsd-restart/webhook-after.txt
cat .epicsd-restart/recovery-summary.json
`},
					Env: map[string]string{
						"EPICSD_HOME": "/tmp/epicsd-home-restart",
					},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"status": "ok"`, `"displayName": "restart-project"`, `"id": "webhook:github:restart-project"`, `"selectedAdapter": "claude"`, `"successes": 2`, `"routeId": "webhook:github:restart-project"`},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.epicsd-restart/status-before.json", MustExist: true, Contains: []string{`"status": "ok"`, `"webhookHTTPAddr": "127.0.0.1:`}},
				{Path: "project/.epicsd-restart/status-after.json", MustExist: true, Contains: []string{`"status": "ok"`, `"webhookHTTPAddr": "127.0.0.1:`}},
				{Path: "project/.epicsd-restart/workspaces-before.json", MustExist: true, Contains: []string{"restart-project", "/workspace/project"}},
				{Path: "project/.epicsd-restart/workspaces-after.json", MustExist: true, Contains: []string{"restart-project", "/workspace/project"}},
				{Path: "project/.epicsd-restart/routes-before.json", MustExist: true, Contains: []string{"webhook:github:restart-project", "\"selectedAdapter\": \"claude\""}},
				{Path: "project/.epicsd-restart/routes-after.json", MustExist: true, Contains: []string{"webhook:github:restart-project", "\"selectedAdapter\": \"claude\""}},
				{Path: "project/.epicsd-restart/runs-after.json", MustExist: true, Contains: []string{`"routeId": "webhook:github:restart-project"`, `"outcome": "succeeded"`}},
				{Path: "project/.epicsd-restart/recovery-summary.json", MustExist: true, Contains: []string{`"successes": 2`, `"routeId": "webhook:github:restart-project"`}},
				{Path: "project/.epicsd-restart/epicsd.log", MustExist: true, Contains: []string{"route=webhook:github:restart-project", "adapter=claude"}},
			},
		},
		{
			Name:         "claude-epicsd-cron-heartbeat-haiku",
			Description:  "Install the original agent-heartbeat Epic, use live Claude to rewrite the installed copy into a 10-second Rust-haiku test profile, then run it through epicsd cron for 60 seconds and collect the real Claude haikus.",
			Tags:         []string{"claude", "live", "daemon", "cron"},
			ImageProfile: "claude",
			RequiredEnv:  []string{"ANTHROPIC_API_KEY"},
			Copies: []CopySpec{
				{From: "e2e/fixtures/claude-web-project", To: "project"},
				{From: "/Users/Testsson/Projects/Kindship/epics/agent-heartbeat", To: "fixtures/agent-heartbeat"},
			},
			Steps: []Step{
				{
					Name:           "install-heartbeat",
					Workdir:        "project",
					Args:           []string{"install", "--host", "claude", "../fixtures/agent-heartbeat"},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Installed Agent Heartbeat for claude into .claude/skills/agent-heartbeat"},
				},
				{
					Name:           "validate-installed-heartbeat",
					Workdir:        "project",
					Args:           []string{"validate", ".claude/skills/agent-heartbeat"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Agent Heartbeat is valid."},
				},
				{
					Name:    "claude-update-installed-heartbeat",
					Program: "claude",
					Workdir: "project",
					Args: []string{
						"-p",
						`Update only the installed Epic at .claude/skills/agent-heartbeat for a live cron test.

Make these exact changes:
1. Rewrite .claude/skills/agent-heartbeat/runtime/state.json so it is valid JSON with:
{
  "state_version": 1,
  "status": "active",
  "name": "Agent Heartbeat",
  "cadence": "10s",
  "last_run": null,
  "next_run": null,
  "current_plan": "runtime/plans/001-rust-haiku.md",
  "nextStep": "Output exactly one fresh three-line haiku about coding in Rust, include the word Rust, use no tools, do not modify files, and stop immediately."
}
2. Create or replace .claude/skills/agent-heartbeat/runtime/plans/001-rust-haiku.md with markdown that instructs every heartbeat run to output exactly one fresh haiku about coding in Rust, in exactly three lines, containing the word Rust, with no title, bullets, fences, or explanation. It must also say: use no tools, inspect nothing, modify no files, and stop immediately after printing the haiku.
3. Update .claude/skills/agent-heartbeat/cron.d/heartbeat.yml so the schedule is "*/10 * * * * *" and the run prompt matches the same Rust-haiku instruction.

Do not modify any other files.
Respond exactly UPDATED`,
						"--dangerously-skip-permissions",
						"--output-format",
						"text",
					},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutEquals:   "UPDATED",
				},
				{
					Name:           "validate-updated-heartbeat",
					Workdir:        "project",
					Args:           []string{"validate", ".claude/skills/agent-heartbeat"},
					ExpectExitCode: 0,
					StdoutContains: []string{"Agent Heartbeat is valid."},
				},
				{
					Name:           "resume-updated-heartbeat",
					Workdir:        "project",
					Args:           []string{"resume", "agent-heartbeat"},
					ExpectExitCode: 0,
					StdoutContains: []string{"runtime/plans/001-rust-haiku.md", "haiku about coding in Rust"},
				},
				{
					Name:    "run-heartbeat-cron",
					Program: "sh",
					Workdir: "project",
					Args: []string{"-lc", `
mkdir -p "$EPICSD_HOME" .epicsd-heartbeat
python3 - <<'PY'
import json
import os
home = os.environ["EPICSD_HOME"]
os.makedirs(home, exist_ok=True)
config = {
    "admin_socket_path": os.path.join(home, "epicsd.sock"),
    "webhook_http_addr": "127.0.0.1:0",
    "max_body_bytes": 1048576,
    "global_queue_capacity": 256,
    "per_workspace_concurrency": 1,
    "dedup_ttl_seconds": 300,
    "scheduler_tick_seconds": 1,
    "allow_insecure_auth_none": False,
    "shutdown_timeout_seconds": 30,
}
with open(os.path.join(home, "config.json"), "w", encoding="utf-8") as fh:
    fh.write(json.dumps(config, indent=2) + "\n")
PY

epicsd > "$EPICSD_HOME/epicsd.log" 2>&1 &
pid=$!
trap 'kill "$pid" >/dev/null 2>&1 || true' EXIT
for i in $(seq 1 50); do
  [ -S "$EPICSD_HOME/epicsd.sock" ] && break
  sleep 0.2
done
[ -S "$EPICSD_HOME/epicsd.sock" ] || { cat "$EPICSD_HOME/epicsd.log"; exit 1; }

epics --json daemon status > .epicsd-heartbeat/status.json
WS=$(epics --json workspace register . --name heartbeat-project | tee .epicsd-heartbeat/workspace.json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
epics --json route upsert --type cron --workspace "$WS" --epic agent-heartbeat --job rust-haiku --cron "*/10 * * * * *" --preferred-adapter claude --auth none --overlap queue_one > .epicsd-heartbeat/route.json
cp "$EPICSD_HOME/config.json" .epicsd-heartbeat/config.json
cp "$EPICSD_HOME/workspaces.json" .epicsd-heartbeat/workspaces.json
cp "$EPICSD_HOME/routes.json" .epicsd-heartbeat/routes.json

sleep 60

ROUTE_ID=$(python3 -c 'import json; print(json.load(open(".epicsd-heartbeat/route.json"))["id"])')
epics --json route disable "$ROUTE_ID" > .epicsd-heartbeat/route-disabled.json

prev=-1
stable=0
for i in $(seq 1 120); do
  epics --json run list --route "$ROUTE_ID" --limit 50 > .epicsd-heartbeat/runs.json
  cp "$EPICSD_HOME/epicsd.log" .epicsd-heartbeat/epicsd.log
  count=$(python3 -c 'import json; runs=json.load(open(".epicsd-heartbeat/runs.json")) or []; print(sum(1 for run in runs if run.get("outcome") == "succeeded"))')
  if [ "$count" = "$prev" ]; then
    stable=$((stable + 1))
  else
    stable=0
    prev="$count"
  fi
  if [ "$count" -ge 3 ] && [ "$stable" -ge 5 ]; then
    break
  fi
  sleep 1
done

python3 - <<'PY'
import json
import re
from pathlib import Path

route = json.load(open(".epicsd-heartbeat/route.json", encoding="utf-8"))
runs = json.load(open(".epicsd-heartbeat/runs.json", encoding="utf-8")) or []
log_lines = Path(".epicsd-heartbeat/epicsd.log").read_text(encoding="utf-8").splitlines()
prefix = f'route={route["id"]} adapter=claude output='
timestamp_re = re.compile(r"^\d{4}/\d{2}/\d{2} ")
haikus = []
for line in log_lines:
    if prefix in line:
        haikus.append([line.split(prefix, 1)[1]])
        continue
    if haikus and not timestamp_re.match(line):
        haikus[-1].append(line)
haiku_text = ["\n".join(part for part in parts if part).strip() for parts in haikus]
haiku_text = [text for text in haiku_text if text]
successful = [run for run in runs if run.get("routeId") == route["id"] and run.get("outcome") == "succeeded" and run.get("adapter") == "claude"]
report = {
    "routeId": route["id"],
    "successfulRuns": len(successful),
    "haikuCount": len(haiku_text),
    "haikus": haiku_text,
}
Path(".epicsd-heartbeat/haikus-report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
markdown = ["# Claude Rust Haikus", ""]
for index, text in enumerate(haiku_text, start=1):
    markdown.append(f"## Haiku {index}")
    markdown.append("")
    markdown.append(text)
    markdown.append("")
Path(".epicsd-heartbeat/haikus-report.md").write_text("\n".join(markdown).rstrip() + "\n", encoding="utf-8")
if len(successful) < 3 or len(haiku_text) < 3:
    raise SystemExit(1)
PY

cat .epicsd-heartbeat/status.json
cat .epicsd-heartbeat/workspace.json
cat .epicsd-heartbeat/route.json
cat .epicsd-heartbeat/runs.json
cat .epicsd-heartbeat/haikus-report.json
cat .epicsd-heartbeat/haikus-report.md
`},
					Env: map[string]string{
						"EPICSD_HOME": "/tmp/epicsd-heartbeat-home",
					},
					PassEnv:        []string{"ANTHROPIC_API_KEY"},
					ExpectExitCode: 0,
					StdoutContains: []string{`"status": "ok"`, `"displayName": "heartbeat-project"`, `"id": "cron:`, `"selectedAdapter": "claude"`, `"successfulRuns":`, `"haikuCount":`, "Claude Rust Haikus", "Rust"},
				},
			},
			Files: []FileAssertion{
				{Path: "project/.claude/skills/agent-heartbeat/runtime/state.json", MustExist: true, Contains: []string{`"cadence": "10s"`, `"current_plan": "runtime/plans/001-rust-haiku.md"`, `"nextStep": "Output exactly one fresh three-line haiku about coding in Rust`}},
				{Path: "project/.claude/skills/agent-heartbeat/runtime/plans/001-rust-haiku.md", MustExist: true, Contains: []string{"Rust", "three lines"}},
				{Path: "project/.claude/skills/agent-heartbeat/cron.d/heartbeat.yml", MustExist: true, Contains: []string{`schedule: "*/10 * * * * *"`, "Rust"}},
				{Path: "project/.epicsd-heartbeat/config.json", MustExist: true, Contains: []string{`"scheduler_tick_seconds": 1`}},
				{Path: "project/.epicsd-heartbeat/route.json", MustExist: true, Contains: []string{`"type": "cron"`, `"selectedAdapter": "claude"`, `"cronExpr": "*/10 * * * * *"`}},
				{Path: "project/.epicsd-heartbeat/runs.json", MustExist: true, Contains: []string{`"routeId": "cron:`, `"outcome": "succeeded"`}},
				{Path: "project/.epicsd-heartbeat/haikus-report.json", MustExist: true, Contains: []string{`"haikuCount":`, `Rust`}},
				{Path: "project/.epicsd-heartbeat/haikus-report.md", MustExist: true, Contains: []string{"# Claude Rust Haikus", "Rust"}},
				{Path: "project/.epicsd-heartbeat/epicsd.log", MustExist: true, Contains: []string{"route=cron:", "adapter=claude"}},
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
