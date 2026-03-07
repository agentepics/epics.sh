# epics.sh

`epics.sh` is the umbrella repo for two related products:

- the `epics.sh` website
- the `epics` Go CLI

The goal is to make Agent Epics easy to publish, discover, install, and run
across AI coding agent CLIs such as Claude Code, Codex, Gemini CLI, and
OpenCode.

## Scope

This repo is intended to hold:

- the public website and docs
- the Go CLI
- the Git-backed Epic registry
- shared schemas and adapter capability metadata
- research and planning docs that drive implementation

The Epic model itself is informed by the sibling `agentepics` reference work,
where an Epic is a valid `SKILL.md` package plus an `EPIC.md` file and optional
runtime surfaces such as `plans/`, `state/`, `log/`, `hooks/`, `cron.d/`, and
`policy.yml`.

## Repository Layout

```text
.
├── README.md
├── AGENTS.md
├── CLAUDE.md
├── go.mod
├── apps/
│   └── web/                  # website app
├── cmd/
│   └── epics/                # Go CLI entrypoint
├── internal/                 # shared Go packages
├── registry/
│   ├── epics/                # registry entries
│   └── schemas/              # metadata and manifest schemas
├── examples/                 # examples and fixtures
└── docs/
    ├── planning/
    ├── architecture/
    ├── specification/
    ├── research/
    └── adapters/
```

## Product Direction

The current direction is:

- one umbrella repo
- one shared registry and schema source of truth
- host-neutral Epic packaging
- host-specific adapters generated around the `epics` CLI

Important explicit objective:

- the Claude adapter should be shippable as an official Claude Code plugin and
  be suitable for Claude's plugin marketplace

## Host Strategy

Initial target posture:

| Host | Current stance |
|---|---|
| Claude Code | strongest first integration target |
| Gemini CLI | strong second integration target |
| OpenCode | dedicated adapter target |
| Codex CLI | partial lifecycle support, strong CLI/MCP target |
| Generic shell-driven agents | baseline support |

The key rule is to use a capability model, not assume every host supports the
same runtime semantics.

## Current Status

This repo is still in the planning and scaffold stage.

Implemented so far:

- repo scaffold
- monorepo directory structure
- planning docs
- daemon/runtime design notes
- adapter research docs

Not implemented yet:

- working website
- working `epics` CLI beyond a stub
- registry schemas
- host adapter code
- CI/release automation

## Docs Map

Start here:

- [Roadmap](./docs/planning/ROADMAP.md)
- [Daemon Architecture](./docs/architecture/DAEMON.md)
- [Spec Extension Notes](./docs/specification/SPEC_EXTENSION.md)
- [Research Snapshot](./docs/research/RESEARCH_SNAPSHOT.md)
- [Adapter Research](./docs/adapters/README.md)

## Development

Current CLI stub:

```bash
go run ./cmd/epics
```

Expected next implementation tracks:

1. build out the Go CLI skeleton under `cmd/epics` and `internal/`
2. initialize the website in `apps/web`
3. define registry and adapter schemas in `registry/schemas/`
4. translate docs into implementation checklists

## Non-goals for V1

- redefining the Epic standard
- pretending all host CLIs have feature parity
- building a full autonomous runtime before the registry and installer are real
- splitting website and CLI into separate repos before ownership and release
  cadence justify it
