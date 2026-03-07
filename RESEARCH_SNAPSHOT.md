# Research Snapshot

This file is a compact persistence layer for the current research work so it
survives context compaction.

Research date: March 7, 2026.

## What Was Produced

Host adapter research was split into:

- `adapters/README.md`
- `adapters/CLAUDE_CODE.md`
- `adapters/CODEX_CLI.md`
- `adapters/GEMINI_CLI.md`
- `adapters/OPENCODE_CLI.md`

Older `INTEGRATION.md` was deleted and replaced by the folder above.

Related planning docs already created earlier:

- `README.md`
- `ROADMAP.md`
- `DAEMON.md`

## Main Conclusions

### Cross-host

- Keep the Epic format host-neutral and file-first.
- Keep `epics` CLI as the canonical behavior surface.
- Use host adapter files/config only for discovery, context injection, command
  wrappers, hooks, MCP wiring, and UX polish.
- If a daemon exists, let `epicsd` own cron, hook dispatch, locks, and runtime
  ledgers instead of encoding runtime semantics separately per host.

### Claude Code

- Strongest current interactive adapter target.
- Best official surfaces: `CLAUDE.md`, hooks, slash commands, subagents,
  settings, MCP.
- Claude Code plugins and marketplace packaging are officially documented and
  should be treated as an explicit `epics.sh` objective.
- Claude-native plugin patterns exist in the wild, but should not define the
  core `epics.sh` model.

### Codex CLI

- Best treated as a strong CLI automation target, not a first-class hook/runtime
  target yet.
- Best surfaces: `AGENTS.md`, `config.toml`, `codex exec`, MCP, sandbox and
  approval controls.
- Community skill-install patterns exist, but should be treated as convenience
  paths rather than the primary strategy.

### Gemini CLI

- Strong first-class adapter target.
- Best surfaces: `GEMINI.md`, custom commands, hooks, settings, extensions,
  Agent Skills, sub-agents, MCP.
- Likely the closest host to Claude Code for Epic lifecycle integration.

### OpenCode

- Broadest programmatic surface.
- Best surfaces: `AGENTS.md`, `opencode.json`, commands, plugins, agents,
  skills, MCP, server modes.
- Probably not the fastest first adapter, but potentially the most flexible long
  term.

## Benchmarks Inspected

### 1. ClaudeClaw

Repo:

- https://github.com/moazbuilds/claudeclaw

Observed strengths:

- Claude-native UX
- plugin packaging
- slash commands
- project-local daemon state
- recurring jobs and dashboard

Observed weakness for `epics.sh`:

- too Claude-specific to define the portable architecture

### 2. Claude-to-IM Skill

Repo:

- https://github.com/op7418/claude-to-im-skill

Observed strengths:

- portable `SKILL.md` packaging baseline
- daemon lifecycle scripts
- `doctor` support
- provider abstraction
- Claude and Codex runtime split

Observed relevance:

- strongest benchmark for adapter architecture

## Primary Source Inventory

### Claude Code

- https://docs.anthropic.com/en/docs/claude-code/settings
- https://docs.anthropic.com/en/docs/claude-code/slash-commands
- https://docs.anthropic.com/en/docs/claude-code/sub-agents
- https://docs.anthropic.com/en/docs/claude-code/hooks
- https://docs.anthropic.com/en/docs/claude-code/memory
- https://docs.anthropic.com/en/docs/claude-code/mcp
- https://code.claude.com/docs/en/plugins

### Codex CLI

- https://github.com/openai/codex
- Local CLI help inspected:
  - `codex --help`
  - `codex mcp --help`
  - `codex exec --help`

### Gemini CLI

- https://geminicli.com/docs/cli/gemini-md
- https://geminicli.com/docs/cli/custom-commands/
- https://geminicli.com/docs/cli/settings
- https://geminicli.com/docs/hooks/
- https://geminicli.com/docs/extensions/
- https://geminicli.com/docs/extensions/reference/
- https://geminicli.com/docs/cli/skills/
- https://geminicli.com/docs/cli/creating-skills/
- https://geminicli.com/docs/cli/cli-reference/

### OpenCode

- https://opencode.ai/docs/rules
- https://opencode.ai/docs/config/
- https://opencode.ai/docs/commands/
- https://opencode.ai/docs/plugins/
- https://opencode.ai/docs/agents/
- https://opencode.ai/docs/skills
- https://opencode.ai/docs/mcp-servers/
- https://opencode.ai/docs/cli/
- https://opencode.ai/docs/acp/

## Outstanding Gaps

- Codex public documentation still appears thinner than the other hosts for
  hook-like lifecycle integration. Treat Codex runtime parity claims carefully.
- Claude plugin/marketplace packaging is visible in the benchmark repo, but the
  most stable official Claude surfaces remain hooks, commands, memory, settings,
  subagents, and MCP.
- If future work needs a definitive host capability matrix, add a machine-
  readable schema under `registry/schemas/` or `docs/`.

## Recommended Next Steps

1. Add a formal host capability schema to the repo.
2. Update `ROADMAP.md` to point at `adapters/`.
3. Turn the adapter docs into implementation checklists for
   `epics host setup <host>`.
4. Decide whether `epicsd` is optional from V1 or deferred entirely.
