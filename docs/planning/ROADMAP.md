# ROADMAP

This roadmap turns `epics.sh` into two deliverables:

- a public Epic directory website
- a Go CLI for Epic installation and host-agent integration

The sequence below is designed to get a credible public MVP live quickly while
leaving room for deeper runtime support later.

## Phase 0: Product and format definition

Goal: lock the product boundary before writing app code.

Deliverables:

- define the `epics.sh` scope relative to the Agent Epics reference spec
- define host support tiers: native, strong, partial, install-only
- define registry metadata schema for listed Epics
- define install manifest schema per host CLI
- define CLI command surface for V1

Key decisions:

- `epics.sh` does not redefine the Epic standard
- the registry is Git-backed first
- public curated sample Epics live in a separate `agentepics/epics` repository
- Codex support is explicitly partial until richer hooks exist
- runtime support is adapter-driven and capability-gated
- Claude Code plugin packaging is an explicit product objective, not an optional
  afterthought

Exit criteria:

- schemas drafted
- support matrix agreed
- CLI scope cut to a realistic V1

## Phase 1: Repository scaffolding

Goal: create a clean monorepo foundation.

Deliverables:

- initialize `apps/web`
- initialize Go module for CLI and internal packages
- add shared docs, schemas, and example content
- define the split between local `examples/`, local `registry/`, and external
  public Epic source repositories
- add lint, test, and release scaffolding
- add CI for website build, CLI test, and registry validation

Suggested structure:

```text
apps/web
cmd/epics
internal/registry
internal/epics
internal/hosts
registry/epics
registry/schemas
examples/
docs/
```

Exit criteria:

- repo builds cleanly
- CI runs on every push
- example registry entries render and validate

## Phase 2: Registry and content model

Goal: make the directory data model real before building polished UI.

Model note:

- `registry/` holds listing metadata, schemas, digests, install metadata, and
  generated indexes
- public curated Epic source repos can live outside this monorepo, starting
  with `agentepics/epics`

Deliverables:

- Epic listing schema
- maintainer schema
- host compatibility schema
- install manifest schema
- digest metadata for reviewed versions
- versioned content layout for `registry/epics/*`
- sample Epic entries based on reference examples

Each Epic listing should include:

- slug
- title
- summary
- tags and category
- maintainers
- source repo
- version
- digest
- host support matrix
- install instructions
- validation status
- screenshots or assets

Exit criteria:

- at least 5 sample entries
- CI schema validation
- generated JSON search index

## Phase 3: Go CLI core

Goal: ship the first useful binary independent of deep host integrations.

V1 commands:

- `epics init`
- `epics validate`
- `epics install`
- `epics remove`
- `epics list`
- `epics info`
- `epics resume`
- `epics export-context`
- `epics doctor`

Implementation priorities:

- Epic package discovery
- schema validation for `SKILL.md`, `EPIC.md`, and registry manifests
- install into workspace-local agent directories
- context export for host prompts and startup hooks
- machine-readable output via `--json`

Exit criteria:

- binary released for macOS, Linux, and Windows
- install and validate flows covered by tests
- works without the website running

## Phase 4: Website MVP

Goal: launch `epics.sh` as a credible public directory.

Core pages:

- landing page
- directory index
- Epic detail page
- docs overview
- CLI install page
- CLI downloads page
- releases or changelog page
- manual landing page
- host compatibility page

Core UX:

- search by tag, host, and category
- filter by support level
- top-of-page copyable `epics install <repo> --agent <optional-agent>` commands
- bootstrap CLI installation when `epics` is not already installed
- per-host install instructions
- copyable CLI snippets
- prominent trust and maintenance metadata

Technical approach:

- Astro + TypeScript + MDX
- static generation from `registry/`
- release-driven generation for CLI downloads and release pages
- simple client-side search from generated JSON

Exit criteria:

- site deploys from CI
- all registry entries have generated detail pages
- install snippets come from source metadata, not hardcoded page content
- reviewed entries expose digest-backed integrity metadata
- CLI download, release, and manual pages are generated from source metadata

## Phase 5: Host adapters

Goal: make the CLI useful inside real agent CLIs.

### 5.1 Claude Code

Target:

- first-class setup via generated `.claude` config
- official Claude Code plugin packaging around the `epics` CLI
- plugin structure designed for marketplace submission
- session-start resume helpers
- stop-time logging helpers
- optional validation hooks on Epic-sensitive writes

Commands:

- `epics host setup claude`
- `epics host doctor claude`

Deliverables:

- Claude plugin directory template
- plugin `commands/`, `hooks/`, and optional `agents/` wrappers that delegate to
  `epics`
- local plugin test flow
- marketplace submission checklist

### 5.2 Gemini CLI

Target:

- equivalent setup flow for `.gemini/settings.json`
- hook-driven resume and logging helpers
- compatibility mapping where Claude and Gemini concepts align

### 5.3 OpenCode

Target:

- adapter or plugin bootstrap for Epic-aware workflows
- install and resume support first
- deeper runtime support only where plugin hooks justify it

### 5.4 Codex

Target:

- install and workspace wiring
- context export and resume helpers
- notify integration if useful
- no misleading claim of full runtime parity

Exit criteria:

- each supported host has generated setup output
- compatibility docs match real adapter behavior
- host-specific smoke tests exist
- Claude adapter works both as direct workspace setup and as a Claude plugin

## Phase 6: Safe state and workflow helpers

Goal: make the CLI operationally useful for live Epics.

Commands:

- `epics state get`
- `epics state set`
- `epics plan list`
- `epics plan current`
- `epics plan create`
- `epics log recent`
- `epics log create`

Behavior requirements:

- preserve unknown fields
- atomic writes
- respect `state/core.json` precedence
- predictable plan numbering
- correct log naming and frontmatter

Exit criteria:

- helpers are test-covered
- direct-use docs exist
- adapters can rely on these helpers

## Phase 7: Runtime-aware features

Goal: support the Epic runtime surface where host capabilities allow it.

Deliverables:

- `epics hooks fire <trigger>`
- `epics cron validate`
- policy loading and diagnostics
- runtime capability reporting per host

Important constraint:

- unsupported runtime features must be surfaced explicitly
- `SKILL.md` guidance must not be presented as equivalent to real runtime
  dispatch

Exit criteria:

- runtime support matrix published
- partial support is visible in both CLI and website

## Phase 8: Publishing and ecosystem workflows

Goal: move from hand-authored registry entries to a repeatable publishing model.

Deliverables:

- author guide for submitting an Epic
- registry contribution templates
- automated validation on pull request
- versioning policy for Epic listings and manifests
- optional generated badges for support level and validation state

Later options:

- submission UI
- private registries
- signed manifests
- verified maintainers

## Cross-cutting work

These tracks should run throughout the roadmap:

- testing: golden tests for CLI output, schema tests, host adapter smoke tests
- documentation: install guides, support matrix, authoring model, troubleshooting
- release engineering: GitHub releases, Homebrew tap or install script, checksums
- design: brand system, docs patterns, directory cards, compatibility labeling

## Recommended MVP Cut

If speed matters, stop after:

- Phase 3 for an internal CLI MVP
- Phase 4 for a public website MVP
- Phase 5.1 and 5.2 for the first strong integrations

That yields a coherent first launch:

- public Epic directory
- installable Go CLI
- strong Claude and Gemini flows
- honest partial support for Codex and OpenCode

## Open Questions

- Should the registry list only curated Epics at first, or also community
  submissions?
- Should installation pull directly from Git repos, downloadable tarballs, or
  registry manifests that point to either?
- Should the website host canonical Epic docs, or mostly point back to source
  repositories?
- How opinionated should `epics host setup` be about modifying user config?
- Do we want one universal install command, or host-specific install commands
  generated from the registry?
