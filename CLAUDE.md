# CLAUDE.md

This is the Claude Code memory file for the `epics.sh` repo.

## Project Identity

`epics.sh` is an umbrella repo for:

- the `epics.sh` website
- the `epics` Go CLI

The repo is intentionally structured as one monorepo rather than separate
website and CLI repos at this stage.

## Main Design Rules

- The Epic format stays spec-first and portable.
- The `epics` CLI is the canonical implementation surface.
- Host integrations should be adapters around the CLI.
- Runtime-heavy concepts belong in optional adapter or daemon layers unless
  deliberately promoted into the standard.

## Important Objectives

- Ship a strong Claude Code adapter.
- Make Claude adapter packaging suitable for the official Claude plugin
  marketplace.
- Keep parity claims honest across Claude, Codex, Gemini CLI, and OpenCode.

## Where Things Live

- `apps/web/` for the website
- `cmd/epics/` for the CLI entrypoint
- `internal/` for shared Go packages
- `registry/` for registry data and schemas
- `docs/` for planning and research

## Read Before Major Changes

- `docs/planning/ROADMAP.md`
- `docs/architecture/DAEMON.md`
- `docs/specification/SPEC_EXTENSION.md`
- `docs/adapters/README.md`

## Current Repo State

This repo is still in scaffold mode.

That means:

- docs are ahead of implementation
- many directories are placeholders
- structural clarity matters more than feature volume right now

## Working Preference

When making early changes:

- prefer small, structural steps
- keep docs and scaffolding consistent
- avoid introducing host-specific assumptions into the core model
