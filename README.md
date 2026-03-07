# epics.sh

`epics.sh` is a website and Go CLI for publishing, discovering, installing, and
running Agent Epics across AI coding agent CLIs.

The goal is simple:

- make Epics easy to browse and trust as reusable packages
- make Epics easy to install into real agent environments
- make Epic support portable across host CLIs with different capabilities

This project is informed by the Agent Epics reference work in the sibling
`agentepics` repo, where an Epic is a valid `SKILL.md` package plus an
`EPIC.md` file and optional runtime surfaces such as `plans/`, `state/`,
`log/`, `hooks/`, `cron.d/`, and `policy.yml`.

## Product Vision

`epics.sh` has two products in one repo:

1. A public website for the Epic directory, documentation, and install flows.
2. A Go CLI that bridges Epic packages into host agents such as Claude Code,
   Gemini CLI, OpenCode, Codex, and generic shell-driven agents.

The website should answer:

- What is an Epic?
- Which Epics exist?
- Is this Epic trustworthy and maintained?
- How do I install it into my agent?
- What level of support does my agent actually have?

The CLI should answer:

- How do I install an Epic into this workspace?
- How do I validate that an Epic is well-formed?
- How do I resume or inspect an Epic safely?
- How do I wire Epic lifecycle helpers into a host CLI?

An explicit objective for the Claude adapter is:

- package the `epics` CLI as an official Claude Code plugin and target Claude's
  plugin marketplace as a supported distribution path

## Core Principles

### 1. Spec first

`epics.sh` should implement the Agent Epic model, not redefine it. The CLI is a
convenience layer for packaging, validation, resumption, and host integration.

### 2. Portable where possible

Every host CLI exposes different extension points. The product should use a
capability model, not pretend every integration is equal.

### 3. Static by default, dynamic where needed

The site should launch from a registry stored in-repo as versioned metadata.
Search, filters, detail pages, manifests, and install snippets can all be
generated from that source of truth. Dynamic submission workflows can come
later.

### 4. File-first registry

Epics are files. The directory should treat each listing as a package with
human docs plus machine-readable metadata and install manifests.

### 5. Honest support matrix

The CLI should clearly distinguish:

- full integration
- partial integration
- install-only support
- unsupported runtime features

## Support Model

The initial host support model should be explicit:

| Host CLI | Initial target | Notes |
|---|---|---|
| Claude Code | Strong support | Best first integration target. Support should include an official Claude Code plugin packaging path and marketplace-ready adapter. |
| Gemini CLI | Strong support | Similar hook model to Claude; good second target. |
| OpenCode | Adapter support | Plugin-driven model needs a dedicated adapter path. |
| Codex | Partial support | Install, validate, resume, and notify-based helpers are realistic today; full hooks depend on future Codex capabilities. |
| Generic CLI | Baseline support | Shell-based install/export flows for any tool that can read files and run commands. |

This matters because "supports Epics" can mean different things:

- can discover and install an Epic
- can load Epic context into a session
- can safely update state, plans, and logs
- can dispatch hooks or recurring jobs
- can enforce policy at runtime

## Planned Website

The website should ship as a content and registry product first.

Core pages:

- home page explaining Epics and the portability story
- directory index with filtering by host support, category, maturity, and tags
- Epic detail pages with screenshots, README excerpts, install commands, and
  compatibility notes
- docs pages for the Epic model and CLI usage
- submit or publish guidance for Epic authors

Core website capabilities:

- registry-backed static generation
- generated JSON index for client-side search
- install snippets per host CLI
- versioned Epic manifests
- trust signals such as version, maintainer, last updated date, and validation
  status

## Planned CLI

The Go CLI should be small, composable, and safe around Epic files.

Initial command surface:

- `epics init`
- `epics validate`
- `epics install`
- `epics remove`
- `epics list`
- `epics info`
- `epics resume`
- `epics export-context`
- `epics host setup <host>`
- `epics doctor`

Second-wave commands:

- `epics state get`
- `epics state set`
- `epics plan list`
- `epics plan current`
- `epics plan create`
- `epics log recent`
- `epics log create`
- `epics hooks fire`

Design constraints:

- preserve unknown fields when editing state
- honor `state/core.json` precedence over `state.json`
- avoid silent downgrades of runtime features
- emit machine-readable output for automation
- keep host adapters isolated behind clear interfaces

Claude-specific objective:

- the Claude adapter should be shippable both as generated workspace setup and
  as a real Claude Code plugin distribution

## Repository Shape

This repo is set up as a small monorepo:

```text
.
├── README.md
├── go.mod
├── apps/
│   └── web/              # website and docs
├── cmd/
│   └── epics/            # Go CLI entrypoint
├── internal/             # Go packages for registry, manifests, hosts, validation
├── registry/
│   ├── epics/            # one folder per listed Epic
│   └── schemas/          # metadata and manifest schemas
├── docs/                 # project docs, not hosted Epic docs
└── examples/             # sample Epic packages and host setups
```

Planning and research docs live under `docs/`.

## Recommended Implementation Approach

### Website stack

Use `Next.js` with TypeScript and MDX.

Reasons:

- handles marketing pages, docs, and directory pages in one app
- can statically generate most pages from the registry
- can expose API routes later for submission, manifests, and search if needed
- makes install snippets, compatibility tables, and registry metadata easy to
  render from typed content

### CLI stack

Use Go for the `epics` binary.

Reasons:

- single static binary distribution
- easy cross-platform releases
- good fit for filesystem-heavy tooling and config generation
- straightforward JSON, YAML, and TOML handling for host integrations

### Registry model

Start with a Git-backed registry in this repo.

Each listed Epic should have:

- metadata file
- manifest for install targets
- markdown content for the directory page
- optional screenshots or assets
- validation output generated in CI

## Non-goals for V1

- building a full autonomous Epic runtime
- claiming identical support across every host agent
- hosting private Epic packages
- building a package marketplace before the registry and installer are solid

## Success Criteria

V1 is successful when:

- the website clearly explains Epics and lists installable packages
- the registry has a clean metadata model and CI validation
- the Go CLI can install and validate Epics reliably
- Claude and Gemini support feel first-class
- Codex and OpenCode have honest, usable adapter flows

## Status

Planning and architecture definition.

See [docs/planning/ROADMAP.md](./docs/planning/ROADMAP.md) for the phased
implementation plan.
