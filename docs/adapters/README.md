# Adapters

This directory replaces the old single `INTEGRATION.md` memo with one file per
host CLI:

- `CLAUDE_CODE.md`
- `CODEX_CLI.md`
- `GEMINI_CLI.md`
- `OPENCODE_CLI.md`

## Research Method

These notes use a simple deep-research workflow:

1. Start from real benchmark projects already integrating with agent CLIs:
   `claudeclaw` and `claude-to-im-skill`.
2. Read the actual project files, not just README claims, to identify concrete
   integration surfaces.
3. Cross-check those patterns against current primary sources for each host CLI:
   official documentation, official repositories, and current CLI help output
   where useful.
4. Separate:
   - officially documented integration methods
   - empirically observed community patterns
   - recommended adapter strategy for `epics.sh`
5. Only ship hosts that can satisfy the same autonomy contract once adapted.

Research date: March 7, 2026.

## Supported-Host Rule

`epics.sh` should only offer hosts that can be brought to the same level of
autonomy through the `epics` CLI and adapter layer. If a host cannot meet that
bar, it remains research only and does not appear as a supported host.

## Cross-Host Summary

| Host | Strongest official surfaces | Main constraint |
|---|---|---|
| Claude Code | `CLAUDE.md`, slash commands, hooks, subagents, MCP, settings | Powerful but Claude-specific distribution patterns can create lock-in |
| Gemini CLI | `GEMINI.md`, custom commands, hooks, extensions, skills, sub-agents, MCP | Extension system is rich, but still host-specific and JSON/TOML-driven |
| OpenCode | `AGENTS.md`, commands, plugins, agents, skills, MCP, config, server modes | Broadest programmatic surface, but more implementation work per integration |

## Recommendation For epics.sh

The common adapter strategy should be:

- keep the Epic package format host-neutral
- keep `epics` CLI as the canonical behavior surface
- use host files only for:
  - discovery
  - instructions/context injection
  - command wrappers
  - lifecycle hooks
  - MCP wiring
- let any future `epicsd` own runtime semantics like cron, hook dispatch,
  locking, and run ledgers
- treat any future `epicsd` as one shared local daemon per OS user account,
  not a repo-local daemon per project

That yields a clean split:

- host adapter = transport and UX
- `epics` CLI = control surface
- `epicsd` = shared local runtime authority across many workspaces
- Epic files = portable state and workflow definition

Codex research remains useful, but Codex should not be presented as a supported
host unless it can meet the same autonomy contract as the supported set.

## v0.5.2 migration checklist

Cross-host adapter work for `0.5.2` should follow these rules:

- discovery still starts from `SKILL.md`
- the standardized `## Agent Epics` footer is portable metadata, not a
  replacement for the operational body
- adapters may consume that footer when generating host-native setup, but they
  should preserve it exactly
- `EPIC.md`, `runtime/`, `hooks/`, `cron.d/`, and `policy.yml` remain the
  canonical epic operating surface
- host-private session IDs, locks, and runtime bookkeeping must never be stored
  in the footer
- legacy footer-less epics must keep working during migration
