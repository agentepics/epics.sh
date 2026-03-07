# Web PRD

Status: Draft

Last updated: 2026-03-07

Owner: `apps/web`

## 1. Product Summary

`epics.sh` web should be two things at once:

- the public directory for discovering Agent Epics
- the official home for the `epics` CLI

The benchmark baseline is `skills.sh`: fast discovery, obvious install CTAs,
detail pages, and hosted docs. `epics.sh` needs parity on those basics, then
extend the product with first-class multi-host install UX, host compatibility
clarity, and a complete CLI home for downloads, releases, changelog, and
manual.

This web product should make Agent Epics easy to:

- discover
- evaluate
- install
- trust
- operate with the `epics` CLI

## 2. Background

The repo already defines `epics.sh` as one umbrella product with two
deliverables: the website and the Go CLI. The website is not a marketing-only
site. It is part of the product surface for discovery, compatibility guidance,
installation flows, and documentation.

Existing repo direction already commits us to:

- a public Epic directory
- docs and compatibility information
- a host-neutral Epic model
- host-specific adapters generated around the `epics` CLI
- honest support labeling rather than pretending all hosts have parity

## 3. Benchmark: `skills.sh`

Benchmark date: 2026-03-07.

Observed current strengths from `skills.sh`:

- directory homepage that immediately explains the product and exposes search
- ranked discovery surfaces such as all-time, trending, and hot listings
- detail pages for packages and individual skills
- install commands shown prominently and consistently
- docs, CLI reference, and FAQ hosted on the same site
- compatibility messaging around supported agents

Relevant benchmark sources:

- homepage: <https://skills.sh/>
- docs: <https://skills.sh/docs>
- CLI reference: <https://skills.sh/docs/cli>
- FAQ: <https://skills.sh/docs/faq>
- example detail page: <https://skills.sh/redis/agent-skills>
- example single-skill page: <https://skills.sh/sourman/skills/skill-create>

What `skills.sh` does not need to solve, but `epics.sh` does:

- richer compatibility truth across multiple agent hosts
- generated host-specific install instructions
- stronger trust metadata around maintainers, validation, and support level
- desktop CLI distribution for macOS, Linux, and Windows
- releases, changelog, and manual for a standalone CLI product

Product implication:

`skills.sh` should be treated as the parity floor for discovery and install UX,
not the ceiling for scope.

## 4. Problem Statement

Today there is no credible public place to browse Agent Epics, compare host
support, understand trust signals, install them quickly, or discover the
official `epics` CLI. The current repo has planning docs and scaffolding, but no
implemented website, no registry schema, and no generated install flows.

Without the website:

- Epics are harder to discover than they should be
- support claims will drift or be overstated
- installation remains fragmented and host-specific
- the CLI has no clear public home or release surface

## 5. Product Vision

`epics.sh` should feel like:

- `skills.sh` for discovery speed and install clarity
- plus a compatibility-aware package directory
- plus the canonical website for the `epics` CLI

A user should be able to arrive at the site and complete one of these jobs in
under a few minutes:

- find an Epic for a task or workflow
- decide whether it works with Claude, Gemini, OpenCode, or Codex
- copy the right install command for their host
- download the `epics` CLI for their platform
- read the manual or changelog when they need depth

## 6. Goals

### 6.1 Primary goals

- Build a credible public Epic directory with feature parity to `skills.sh` on
  core discovery and install flows.
- Make installation extremely easy with obvious, copyable, host-specific
  commands generated from source metadata.
- Establish `epics.sh` as the official home of the `epics` CLI.
- Present host support honestly using capability-based labeling and visible
  caveats.
- Make trust visible through validation state, maintainer identity, versioning,
  source links, and maintenance metadata.

### 6.2 Secondary goals

- Provide a docs/manual experience that can scale with the CLI and adapter work.
- Create a clean path from directory discovery to CLI installation to host
  setup.
- Leave room for richer ecosystem features later, such as submissions, badges,
  and private registries.

## 7. Non-Goals For V1

- No claim that all hosts support the same runtime semantics.
- No web-only install instructions that diverge from CLI or registry metadata.
- No runtime dashboard or daemon UI.
- No community submission workflow in the first public release.
- No requirement that the site be dynamic or database-backed for V1.

## 8. Target Users

### 8.1 Epic users

Developers using AI coding agents who want to find reusable workflows, install
them quickly, and understand whether they work in their current host.

### 8.2 Epic authors and maintainers

People publishing Epics who need a credible listing surface, accurate support
presentation, and a clear path to installation and validation.

### 8.3 CLI adopters

Users who may arrive looking for the `epics` CLI first and need platform
downloads, release notes, docs, and installation guidance.

## 9. Product Principles

- CLI canonical: the website reflects the real `epics` behavior surface rather
  than inventing a separate one.
- Host-neutral core: Epic packaging stays portable; host-specific UX is layered
  on top.
- Honest compatibility: partial support is shown plainly, especially for Codex.
- Install-first UX: every major page should create a short path to action.
- Source-derived content: install snippets and compatibility data come from
  registry or release metadata, not hardcoded copy.
- Static-first delivery: prefer static generation and generated indexes before
  introducing server complexity.

## 10. Requirements

### 10.1 Website MVP pages

The MVP must include:

- landing page
- directory index
- Epic detail page
- docs overview
- CLI install page
- host compatibility page
- CLI downloads page
- releases page
- changelog page
- manual landing page

### 10.2 Landing page

The landing page must:

- explain what Agent Epics are and why they matter
- explain what the `epics` CLI is
- offer immediate search or browse entry into the directory
- provide a primary install CTA for the CLI
- provide a secondary CTA for browsing Epics
- show host support posture at a glance
- feature curated or popular Epics once ranking data exists

### 10.3 Directory index

The directory index must:

- support search by title, summary, tag, category, and host
- support filtering by support level
- support sorting by at least featured and newest in V1
- leave room for trending or popular ranking once telemetry or installs exist
- show strong summary cards with title, summary, tags, support badges, and
  trust signals

### 10.4 Epic detail page

Each Epic detail page must show:

- title and summary
- what the Epic is for
- tags and category
- source repository
- version
- maintainers
- host support matrix
- support caveats by host
- generated install instructions per host
- validation state
- screenshots or assets where available
- links to source docs

Each detail page must include:

- a default install CTA at the top of the page
- a copyable `epics install <repo> --agent <optional-agent>` command at the top
  of the page, similar to the install treatment on `skills.sh`
- host-specific copyable snippets
- a clear explanation when support is partial or install-only

Install presentation requirements:

- the default install command should use the Epic source repo path
- the `--agent` flag should appear when a host-specific install path is being
  shown
- the top-of-page install surface should prioritize speed over explanation
- more detailed setup guidance can appear below the fold

### 10.5 Compatibility page

The compatibility page must:

- define support levels such as native, strong, partial, and install-only
- explain how support differs by host capability
- compare Claude Code, Gemini CLI, OpenCode, Codex, and generic shell agents
- align exactly with adapter reality and published docs

### 10.6 CLI home

The CLI product surface must include:

- platform downloads for macOS, Linux, and Windows
- install instructions
- release channels if we introduce them
- changelog
- release notes
- manual and command reference
- quickstart for `epics init`, `install`, `validate`, and `host setup`

The CLI pages should make the project feel like a real downloadable developer
tool, not just a code repository.

### 10.7 Docs and manual

The docs surface must include:

- product overview
- getting started
- install guide
- host compatibility guide
- authoring model
- troubleshooting
- CLI manual

The docs experience should support MDX and relative links so the docs can grow
without changing architecture later.

### 10.8 Install UX

Install UX is a top-level requirement.

The site must make installation extremely easy by:

- showing copyable commands in the first viewport where possible
- defaulting to the right install path based on selected host
- generating snippets from registry or release metadata
- distinguishing CLI installation from Epic installation
- minimizing the number of decisions before a user can try the product

Recommended install model:

- direct Epic install on detail pages uses `epics install <repo> --agent
  <optional-agent>`
- if `epics` is already installed, that direct command is the primary path
- if `epics` is not installed, the site should also offer a bootstrap one-liner
  that installs the CLI first and then runs the requested `epics install`
  command

Recommended bootstrap shape:

- Unix-like systems: `curl -fsSL https://epics.sh/install.sh | sh`
- Windows: PowerShell bootstrap equivalent
- after bootstrap, the site should return the user to the canonical `epics
  install ...` command rather than creating a permanently separate install path

This keeps the visible product model simple:

- install the CLI
- run `epics install ...`

Potential future convenience path:

- an `npx` wrapper could exist later, but should not be the primary V1 path for
  a Go CLI unless it is proven simpler and equally trustworthy

### 10.9 Trust and quality signals

The site must surface:

- validation state
- maintainer identity
- source repository
- last updated date
- version
- support level by host
- warnings for partial support

If security review or verification programs exist later, the UI should be able
to adopt them without redesign.

### 10.10 Integrity and install validation

The product must support integrity validation for reviewed Epic versions.

Each listed Epic version should include digest metadata so installation can
verify that the downloaded remote repo content matches the reviewed and
presented version on `epics.sh`.

Requirements:

- registry metadata should include a digest for the reviewed installable source
- `epics install` should be able to validate remote content against that digest
- the website should surface the reviewed version and digest-backed validation
  status
- integrity validation should work without implying that every repo reference is
  mutable or equally trusted

## 11. Information Architecture

Top-level navigation should likely include:

- Directory
- CLI
- Docs
- Compatibility
- Changelog

Suggested route model:

- `/`
- `/epics`
- `/epics/[slug]`
- `/cli`
- `/cli/install`
- `/cli/downloads`
- `/cli/releases`
- `/cli/changelog`
- `/docs`
- `/docs/[...slug]`
- `/compatibility`

## 12. Data And Content Dependencies

The web product depends on work outside `apps/web`.

Required upstream inputs:

- Epic listing schema
- maintainer schema
- host compatibility schema
- install manifest schema
- digest metadata for reviewed Epic versions
- sample registry entries
- generated JSON search index
- release metadata for CLI downloads
- changelog source
- docs/manual content

Content sourcing note:

- sample Epics authored by `epics.sh` should live in a separate repository under
  the `agentepics` organization rather than inside this monorepo

Implication:

`apps/web` can be scaffolded now, but meaningful end-to-end value depends on the
registry and CLI metadata existing in machine-readable form.

## 13. Technical Direction

Current planned stack remains correct:

- Astro
- TypeScript
- MDX
- static generation from `registry/`
- simple client-side search from generated JSON

Why Astro:

- the product is primarily a static directory, docs/manual site, and release
  surface
- MDX is a first-class requirement
- low-JS delivery is a better default than a heavier app framework for this V1
- interactive pieces such as search, filters, and copy/install widgets can be
  handled as focused islands

Additional technical expectations for the web app:

- generated pages from registry data
- generated install snippets from source metadata
- generated release/download pages from release metadata
- a design system that can express host support clearly
- strong mobile behavior for install and docs flows

## 14. Success Criteria

The website MVP is successful when:

- a new user can discover an Epic and copy an install command in under 2 minutes
- the site clearly communicates host support without misleading parity claims
- all registry entries have generated detail pages
- CLI download and installation paths are obvious for macOS, Linux, and Windows
- docs and manual are navigable without leaving the site
- the site deploys from CI

## 15. Phasing

### Phase A: PRD and data contracts

- finalize product requirements
- finalize registry and compatibility schemas
- define release metadata inputs for CLI pages

### Phase B: Website MVP foundation

- scaffold `apps/web`
- build landing page, directory, detail page, docs shell, and compatibility page
- wire static generation and search index

### Phase C: CLI home

- build downloads, releases, changelog, and manual surfaces
- ensure install and docs flows connect cleanly to the directory

### Phase D: Post-MVP enhancements

- curated featured Epics
- trending or popular ranking
- submission flows
- badges and richer verification

## 16. Decisions

### 16.1 Homepage strategy

Decision:

- use a split-hero homepage with discovery-led emphasis and an equally obvious
  CLI install path

Reason:

- `epics.sh` is both a directory and a CLI home, so the homepage must support
  both jobs without reducing the site to either a marketing page or a download
  page

### 16.2 Ranking strategy for first public release

Decision:

- ship curated featured items and `newest`
- defer trending, hot, and all-time ranking until real usage signals exist

Reason:

- ranking without durable data creates false precision and weakens trust

### 16.3 Manual source of truth

Decision:

- manual content should be canonical in repo source files and rendered on the
  website

Reason:

- one source of truth avoids drift between the CLI and the website while still
  allowing the web app to be the best reading experience

### 16.4 Releases and changelog source model

Decision:

- downloadable artifacts should come from GitHub releases
- changelog content should be canonical in repo files
- the website should merge both into one coherent release surface

Reason:

- this matches the natural ownership of binaries versus versioned release notes

### 16.5 Minimum trustworthy metadata

Decision:

- Epic cards should require at minimum:
  - `slug`
  - `title`
  - `summary`
  - `category`
  - `tags`
  - `source_repo`
  - `maintainers`
  - `version`
  - per-host support levels
  - `last_updated`
- Epic detail pages should additionally require:
  - install metadata for `epics install <repo> --agent <optional-agent>`
  - validation status
  - host-specific caveats

Reason:

- title and summary alone are not sufficient for a trust-oriented directory

### 16.6 Sample Epic sourcing

Decision:

- sample Epics authored and maintained by `epics.sh` should live in a separate
  repository under the `agentepics` organization

Suggested repo name:

- `agentepics/epics`

Alternatives:

- `agentepics/epics-examples`
- `agentepics/epics-catalog`

Recommendation:

- use `agentepics/epics` to mirror the benchmark pattern from
  `anthropics/skills`: one repo for the public, curated package set and another
  repo for the spec itself
