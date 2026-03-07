# AGENTS.md

This repo is the umbrella repository for the `epics.sh` website and the `epics`
CLI.

## Working Rules

- Keep the Epic format host-neutral. Do not encode Claude-, Codex-, Gemini-, or
  OpenCode-specific semantics into the portable core model unless explicitly
  documented as adapter/runtime-only.
- Treat `epics` CLI as the canonical behavior surface. Host integrations should
  wrap or call the CLI rather than reimplementing core behavior.
- Treat daemon/runtime concepts as conditional. The portable baseline should
  remain valid without `epicsd`.
- Keep docs aligned with the actual repo structure. This repo should not contain
  local absolute filesystem paths in committed docs.

## Repo Shape

- `apps/web/` contains the website
- `cmd/epics/` contains the CLI entrypoint
- `internal/` contains shared Go packages
- `registry/` contains registry data and schemas
- `docs/` contains planning, research, adapter, architecture, and spec notes

## Priority Docs

Read these before making architectural changes:

- `docs/planning/ROADMAP.md`
- `docs/architecture/DAEMON.md`
- `docs/specification/SPEC_EXTENSION.md`
- `docs/adapters/README.md`

## Current Priorities

- establish the CLI structure
- establish the web app structure
- define registry schemas
- define host capability matrices
- preserve a clean split between portable Epic concepts and adapter/runtime
  concepts

## Adapter Principles

- Claude Code: strong first target, including official plugin packaging
- Gemini CLI: strong first-class adapter target
- Codex CLI: treat as CLI/MCP-first, not hook-first
- OpenCode: broad programmatic surface, but higher integration complexity

## Daemon Principles

If `epicsd` is implemented, assume:

- ingress routing must be cross-adapter
- route binding, adapter selection, and executor binding are separate layers
- runtime metadata should stay separate from the portable Epic files

## Documentation Hygiene

- prefer relative links
- do not reference local absolute paths
- update README and related docs when moving files or changing repo structure
