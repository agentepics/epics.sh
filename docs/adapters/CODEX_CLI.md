# Codex CLI Adapter

## Summary

Codex CLI has a real integration surface, but it is currently thinner and less
host-opinionated than Claude Code or Gemini CLI.

The strongest current methods are:

- `AGENTS.md`
- `config.toml` plus profiles and `-c` overrides
- non-interactive execution modes
- MCP client and server modes
- sandbox and approval controls

The main implication for `epics.sh` is that Codex should be treated as a strong
CLI automation target, but only a partial lifecycle-hook target today.

## Official and Observed Integration Methods

### 1. `AGENTS.md`

Codex uses `AGENTS.md` as an instruction surface. This is visible in:

- the official `openai/codex` repository itself, which includes `AGENTS.md`
- current Codex behavior in this environment
- community usage around project-level Codex guidance

For `epics.sh`, this is the natural place to inject:

- workspace Epic conventions
- preferred `epics` commands
- resume guidance
- cross-repo operating rules

Unlike Claude Code, Codex does not currently offer the same breadth of
documented host-native wrapper systems around this file, so `AGENTS.md` is even
more central.

### 2. `config.toml`, profiles, and per-run overrides

The current Codex CLI help surface shows:

- `~/.codex/config.toml`
- config profiles via `--profile`
- dotted config overrides via `-c key=value`

This is the main official mechanism for:

- model selection
- sandbox mode
- approval policy
- environment behavior
- feature flags

For `epics.sh`, this suggests a good adapter strategy:

- avoid mutating global config unless explicitly requested
- generate profile snippets or local install instructions
- prefer workspace-local guidance plus command wrappers over deep global changes

### 3. Non-interactive execution modes

Current Codex CLI help exposes:

- `codex exec`
- `codex review`
- `codex resume`
- `codex fork`
- JSONL output for `exec`
- schema-constrained output

This is a strong integration surface for `epics.sh`, especially for:

- CI or automation mode
- explicit Epic validation runs
- resume-context export
- machine-readable workflows

This is one of Codex’s biggest strengths for Epic tooling.

### 4. MCP client and server modes

Current Codex CLI help exposes:

- `codex mcp add/list/get/remove/login/logout`
- `codex mcp-server`

This means Codex can integrate with `epics.sh` in two directions:

- Codex consuming Epic-aware MCP services
- Codex exposing itself to other automation surfaces via MCP server mode

For future `epics.sh`, an MCP server is one of the cleanest ways to offer:

- Epic discovery
- Epic status
- structured plan/state/log operations
- runtime capability inspection

### 5. Sandbox and approval controls

Current Codex CLI help exposes:

- `--sandbox` modes
- `--ask-for-approval`
- `--full-auto`
- `--dangerously-bypass-approvals-and-sandbox`

These are not hooks, but they are still important adapter inputs because they
shape what unattended Epic automation can realistically do inside Codex.

For `epics.sh`, Codex integration should treat these as execution policy
constraints, not as equivalent to lifecycle extensibility.

## Community and Benchmark Patterns

### Claude-to-IM Skill

`claude-to-im-skill` is the most relevant Codex benchmark in this project set.
It uses:

- a shared `SKILL.md` package
- a separate Codex install script
- a Codex-specific provider implementation
- runtime selection among `claude`, `codex`, and `auto`

The important lesson is not the IM bridge itself. It is the adapter style:

- portable package first
- host-specific bootstrap second
- provider abstraction for real runtime differences

### `~/.codex/skills` as a community convention

The benchmark project installs into `~/.codex/skills/`, but this should be
treated carefully.

This is a useful empirical pattern for Codex-compatible distribution, but it is
not the strongest officially documented Codex surface in the way `AGENTS.md`,
`config.toml`, or MCP are.

For `epics.sh`, this means:

- okay to support as a convenience path
- not okay to build the entire Codex strategy around it

## Other Possible Codex CLI Integration Methods

### Method A: `AGENTS.md` + `epics` CLI

Good for:

- minimal installation
- portable project guidance
- no dependence on advanced host features

This should be the baseline Codex adapter.

### Method B: non-interactive automation via `codex exec`

Good for:

- CI pipelines
- batch Epic validation
- generating structured summaries
- deterministic wrappers around Epic workflows

This is a stronger Codex path than trying to emulate Claude-style hook-heavy
integration.

### Method C: MCP-backed integration

Good for:

- structured tools
- future daemon bridge
- capability discovery

This is probably the best advanced Codex path.

### Method D: profile-based operational presets

Good for:

- documented sandbox and approval presets
- team-standard execution modes
- safer onboarding

Example targets:

- `epics-safe`
- `epics-automation`
- `epics-readonly`

## Main Constraint

Codex still appears weaker than Claude Code and Gemini CLI for direct
hook-style lifecycle integration.

That means `epics.sh` should avoid promising:

- full runtime parity
- automatic condition-triggered Epic behavior entirely inside Codex
- host-native cron orchestration without extra runtime help

Instead, Codex should excel at:

- instruction loading
- CLI automation
- MCP-based extensions
- explicit `epics` command execution

## Recommended Adapter Strategy For epics.sh

Phase order:

1. `AGENTS.md` guidance + documented install flow
2. `epics` CLI as the operational surface
3. `codex exec` workflows for validation and automation
4. MCP-backed advanced integration
5. optional daemon bridge for runtime features beyond Codex’s native lifecycle

## Sources

- OpenAI Codex official repository:
  https://github.com/openai/codex
- Local Codex CLI help surface inspected on March 7, 2026:
  `codex --help`
- Local Codex CLI MCP help surface inspected on March 7, 2026:
  `codex mcp --help`
- Local Codex CLI exec help surface inspected on March 7, 2026:
  `codex exec --help`
- Benchmark repo:
  https://github.com/op7418/claude-to-im-skill
