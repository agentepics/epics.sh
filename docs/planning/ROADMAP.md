# ROADMAP

This roadmap turns `epics.sh` into two deliverables:

- a public Epic directory website
- a Go CLI for Epic installation and host-agent integration

The sequence below is designed to get a credible public MVP live quickly while
leaving room for deeper runtime support later.

## Status Snapshot

Last updated: 2026-03-08

Current implementation status:

- Phase 0: complete
- Phase 1: complete
- Phase 2: partial
  Schema validation exists, but generated registry index work is still pending.
- Phase 3: complete
  The CLI ships `init`, `validate`, `install`, `info`, `resume`, `doctor`,
  and `host setup`. `doctor` now distinguishes authored Epics from installed
  Epics and warns on missing or drifted local install sources.
- Phase 5: partial
  Claude, Gemini, and OpenCode workspace setup are implemented, and `epics
  host doctor <host>` now exists for all supported hosts. Claude plugin
  packaging, hooks, and richer runtime integration are still pending. A
  retained Claude live-chat harness also exists for qualitative adapter
  feedback in the real live-test container.
- Phase 6: complete
  `state`, `status`, `plan`, and `log` helper commands are implemented and
  test-covered.
- Phase 7: partial
  `epics cron validate` is implemented. Hooks, policy loading, and runtime
  capability reporting are still pending.

## Phase 0: Product and format definition

Goal: lock the product boundary before writing app code.

Deliverables:

- define the `epics.sh` scope relative to the Agent Epics reference spec
- define the supported-host contract for `epics.sh`
- define registry metadata schema for listed Epics
- define install manifest schema per host CLI
- define CLI command surface for V1

Key decisions:

- `epics.sh` does not redefine the Epic standard
- the registry is Git-backed first
- public curated sample Epics live in a separate `agentepics/epics` repository
- only hosts that satisfy the autonomy contract should be offered
- runtime support is adapter-driven but should converge on one autonomy promise
- any future `epicsd` should default to one shared local daemon per OS user,
  not one daemon per project
- Claude Code plugin packaging is an explicit product objective, not an optional
  afterthought

Exit criteria:

- schemas drafted
- supported-host contract agreed
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
- supported-host schema
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
- install instructions
- validation status
- screenshots or assets

Exit criteria:

- at least 5 sample entries
- CI schema validation
- generated JSON search index

## Phase 3: Go CLI core

Goal: ship the first useful binary independent of deep host integrations.

Status: complete

V1 commands:

- `epics init`
- `epics validate`
- `epics install`
- `epics info`
- `epics resume`
- `epics doctor`
- `epics host setup claude`

Additional implemented commands beyond the original V1 cut:

- `epics host setup gemini`
- `epics host setup opencode`
- `epics host doctor <host>`
- `epics state get`
- `epics state set`
- `epics plan list`
- `epics plan current`
- `epics plan create`
- `epics log recent`
- `epics log create`
- `epics cron validate`

Additional implemented behavior:

- `epics doctor` distinguishes authored Epics from installed Epics
- `epics doctor` reports missing local install sources and source drift warnings

Implementation priorities:

- Epic package discovery
- schema validation for `SKILL.md`, `EPIC.md`, and registry manifests
- install into workspace-local agent directories
- prompt for host choice when `--host` is omitted instead of auto-detecting
- keep `.epics/` for CLI metadata while using host-local package paths as the
  canonical install location
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
- supported agents page

Core UX:

- search by tag and category
- top-of-page copyable installer-led commands that pass an Epic path
- bootstrap CLI installation when `epics` is not already installed
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

Status: partial

### 5.1 Claude Code

Target:

- first-class setup via generated `.claude` config and project-local skill
  installs under `.claude/skills/<slug>/`
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
- host-local install target rooted in `.claude/skills/<slug>/`
- plugin `commands/`, `hooks/`, and optional `agents/` wrappers that delegate to
  `epics`
- local plugin test flow
- marketplace submission checklist

Current status:

- workspace setup under `.claude/skills/<slug>/` is implemented
- generated `.claude/commands/` wrappers are implemented
- additive `CLAUDE.md` guidance injection is implemented
- `epics host doctor claude` is implemented
- a retained live-chat test harness exists for real Claude conversations inside
  the live container image
- plugin packaging, hooks, and marketplace prep are still pending

### 5.2 Gemini CLI

Target:

- equivalent setup flow for `.gemini/settings.json`
- hook-driven resume and logging helpers
- compatibility mapping where Claude and Gemini concepts align

Current status:

- project-local setup under `.gemini/skills/<slug>/` is implemented
- generated `.gemini/commands/` wrappers are implemented
- additive `GEMINI.md` guidance injection is implemented
- `epics host doctor gemini` is implemented
- settings and hook generation are still pending

### 5.3 OpenCode

Target:

- adapter or plugin bootstrap for Epic-aware workflows
- install and resume support first
- deeper runtime support only where plugin hooks justify it

Current status:

- project-local setup under `.opencode/skills/<slug>/` is implemented
- generated `.opencode/commands/` wrappers are implemented
- additive `AGENTS.md` guidance injection is implemented
- `epics host doctor opencode` is implemented
- plugin/config bootstrap beyond workspace setup is still pending

Exit criteria:

- each supported host has generated setup output
- `epics install` never guesses the host; it uses `--host` or a prompt
- supported-agent docs match real adapter behavior
- host-specific smoke tests exist
- Claude adapter works both as direct workspace setup and as a Claude plugin

## Phase 6: Safe state and workflow helpers

Goal: make the CLI operationally useful for live Epics.

Status: complete

Commands:

- `epics state get`
- `epics state set`
- `epics status`
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

Status: partial

Deliverables:

- `epics hooks fire <trigger>`
- `epics cron validate`
- policy loading and diagnostics
- runtime capability reporting per host

Important constraint:

- unsupported runtime features must be surfaced explicitly
- `SKILL.md` guidance must not be presented as equivalent to real runtime
  dispatch

Current status:

- `epics cron validate` is implemented and test-covered
- `epics hooks fire <trigger>` is still pending
- policy loading and diagnostics are still pending
- runtime capability reporting per host is still pending

Daemon direction:

- a future Phase B `epicsd` should be a shared local daemon per OS user account
- that daemon should coordinate many workspaces and installed Epics for the same
  user
- per-workspace and per-Epic isolation should be enforced inside daemon state,
  not by running one daemon per project
- Phase C daemon-backed adapters should target that same shared daemon model

Exit criteria:

- supported-host contract published
- unsupported hosts are omitted from both CLI and website

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

- testing: golden tests for CLI output, schema tests, host adapter smoke tests,
  and live host chat transcripts for qualitative adapter feedback
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
- honest partial support for OpenCode and future hosts

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
