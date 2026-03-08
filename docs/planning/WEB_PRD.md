# Web PRD

Status: Draft

Last updated: 2026-03-08

Owner: `apps/web`

## 1. Product Summary

`epics.sh` web should be two things at once:

- the public directory for discovering Agent Epics
- the official home for the `epics` CLI

The benchmark baseline is `skills.sh`: fast discovery, obvious install CTAs,
detail pages, and hosted docs. `epics.sh` needs parity on those basics, then
extend the product with first-class install UX and a complete CLI home for
downloads, releases, changelog, and manual.

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
- docs and supported-host information
- a host-neutral Epic model
- host-specific adapters generated around the `epics` CLI
- only offering hosts that can satisfy the same autonomy contract

## 3. Benchmark: `skills.sh`

Benchmark date: 2026-03-07. Deep design analysis: 2026-03-08.

### 3.1 Observed strengths

- directory homepage that immediately explains the product and exposes search
- ranked discovery surfaces such as all-time, trending, and hot listings
- detail pages for packages and individual skills
- install commands shown prominently and consistently
- docs, CLI reference, and FAQ hosted on the same site
- compatibility messaging around supported agents

### 3.2 Benchmark sources

- homepage: <https://skills.sh/>
- docs: <https://skills.sh/docs>
- CLI reference: <https://skills.sh/docs/cli>
- FAQ: <https://skills.sh/docs/faq>
- audits: <https://skills.sh/audits>
- example detail page: click any skill on the homepage leaderboard

### 3.3 Information architecture analysis

`skills.sh` has exactly three page types:

1. **Homepage** — compact identity zone (~200px) then leaderboard/directory
2. **Docs** — sidebar with 3 tabs (Overview, CLI, FAQ), prose content
3. **Audits** — security audit table with color-coded results
4. **Detail pages** — per-skill pages reached from the leaderboard

Total navigation links: 2 (Audits, Docs). No separate directory link because
the homepage IS the directory. No separate CLI page because CLI docs fold into
Docs.

### 3.4 Design goals inferred from the implementation

**The homepage is the product.** The brand zone (ASCII art, one sentence, install
command, agent logos) occupies ~200px and exists only so users know they arrived
at the right place. Below that, the leaderboard IS the page. There is no
separate directory page, no hero section in the traditional sense.

**Every page terminates at an install command.** Homepage shows CLI install.
Detail pages show skill-specific install. Docs show install in "Getting
Started." The install command is the universal primary CTA.

**Ruthless navigation pruning.** Only pages that serve a unique purpose exist.
"Directory" is not a link because you are already on it. "CLI" is not a link
because CLI docs live inside Docs. "Compatibility" is not a link because
compatibility shows per-skill on detail pages. "Changelog" does not exist as a
page.

**Social proof drives discovery.** Install counts are the primary metadata.
Leaderboard is sorted by installs. Trending and Hot tabs create urgency.
Per-skill detail pages show install breakdowns by agent. No editorial curation
or "featured" badge — popularity IS the ranking.

**Detail pages are functional, not promotional.** Breadcrumb, install command,
rendered SKILL.md content, metadata sidebar (installs, repo, dates, security
audits, per-agent installs). The skill's own content speaks for itself.

**No footer.** Pages just end. No secondary nav, no redundant links.

### 3.5 What `skills.sh` does not need to solve, but `epics.sh` does

- a stricter supported-host contract across multiple agent CLIs
- generated installer-led install instructions
- stronger trust metadata around maintainers, validation, and digest review
- desktop CLI distribution for macOS, Linux, and Windows
- releases, changelog, and manual for a standalone CLI product

### 3.6 Product implication

`skills.sh` should be treated as the parity floor for discovery and install UX,
not the ceiling for scope. However, the information architecture lesson is
clear: merge pages that don't need to be separate, and make the directory the
homepage rather than a page you navigate to.

### 3.7 Implications for `epics.sh` IA

`epics.sh` has an additional objective that `skills.sh` does not: it is also the
canonical home for the `epics` CLI as a downloadable developer tool. This means
`epics.sh` legitimately needs a CLI surface (downloads, releases, changelog)
that `skills.sh` does not have.

Recommended IA adaptation:

| `skills.sh` page | `epics.sh` equivalent | Notes |
|---|---|---|
| Homepage (leaderboard) | Homepage (directory) | Homepage IS the directory |
| — | — | No separate `/epics` index needed |
| Detail page | `/epics/[slug]` | Same pattern |
| Docs (3 tabs) | `/docs` (sidebar nav) | Fold compatibility into docs |
| Audits | Defer to post-MVP | Not needed until trust program exists |
| — | `/cli` | CLI home: downloads, changelog, manual |

Recommended navigation: 2-3 links max (Docs, CLI, and possibly Changelog if
it cannot live inside CLI).

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
- plus a supported-host-aware package directory
- plus the canonical website for the `epics` CLI

A user should be able to arrive at the site and complete one of these jobs in
under a few minutes:

- find an Epic for a task or workflow
- trust that it works on every supported host
- copy the install command
- download the `epics` CLI for their platform
- read the manual or changelog when they need depth

## 6. Goals

### 6.1 Primary goals

- Build a credible public Epic directory with feature parity to `skills.sh` on
  core discovery and install flows.
- Make installation extremely easy with obvious, copyable bootstrap and install
  commands generated from source metadata.
- Establish `epics.sh` as the official home of the `epics` CLI.
- Present a clear supported-host contract: every offered host meets the same
  autonomy promise.
- Make trust visible through validation state, maintainer identity, versioning,
  source links, and maintenance metadata.

### 6.2 Secondary goals

- Provide a docs/manual experience that can scale with the CLI and adapter work.
- Create a clean path from directory discovery to CLI installation to host
  setup.
- Leave room for richer ecosystem features later, such as submissions, badges,
  and private registries.

## 7. Non-Goals For V1

- No support for hosts that cannot meet the autonomy contract.
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
- Supported-host contract: only hosts that can uphold the autonomy promise are
  offered.
- Install-first UX: every major page should create a short path to action.
- Source-derived content: install snippets and supported-host data come from
  registry or release metadata, not hardcoded copy.
- Static-first delivery: prefer static generation and generated indexes before
  introducing server complexity.

### 9.1 Design and editorial direction

- The visual direction should feel terminal-like and operational, in the spirit
  of `skills.sh`, without cloning its layout or details one-to-one.
- Typography, spacing, and surface treatment should suggest a serious developer
  tool rather than a generic SaaS landing page.
- Copy should borrow themes from `agentepics/agentepics`: long-running work,
  continuity across sessions, portable workflow packages, and durable state.
- Product copy should avoid hype and stay concrete about what Epics preserve:
  plans, checkpoints, logs, decisions, resume context, and host-specific setup.

## 10. Requirements

### 10.1 Website MVP pages

The MVP must include:

- landing page
- directory index
- Epic detail page
- docs overview
- CLI install page
- supported agents page
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
- show supported agents at a glance
- feature curated or popular Epics once ranking data exists

### 10.3 Directory index

The directory index must:

- support search by title, summary, tag, category, and host
- support filtering by category in V1
- support sorting by at least featured and newest in V1
- leave room for trending or popular ranking once telemetry or installs exist
- show strong summary cards with title, summary, tags, and trust signals

### 10.4 Epic detail page

Each Epic detail page must show:

- title and summary
- what the Epic is for
- tags and category
- source repository
- version
- maintainers
- generated install instructions, with host-specific variants where needed
- validation state
- screenshots or assets where available
- links to source docs

Each detail page must include:

- a default install CTA at the top of the page
- a copyable bootstrap-and-install command that passes the Epic path directly to
  the installer
- digest-backed trust metadata

Install presentation requirements:

- the default install command should use the Epic source repo path as an
  installer argument
- the default install command should not force an upfront agent choice
- the installer should ask the user a short series of setup questions after it
  starts
- the current project folder should be the install context; no global Epic
  install location should be implied
- the top-of-page install surface should prioritize speed over explanation
- more detailed setup guidance can appear below the fold

### 10.5 Supported agents page

The supported agents page must:

- define the supported-host contract
- list the hosts that satisfy it
- make clear that unsupported hosts are omitted rather than downgraded
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
- supported agents guide
- authoring model
- troubleshooting
- CLI manual

The docs experience should support MDX and relative links so the docs can grow
without changing architecture later.

### 10.8 Install UX

Install UX is a top-level requirement.

The site must make installation extremely easy by:

- showing copyable commands in the first viewport where possible
- defaulting to a guided install path and asking host questions when `--host` is
  not already provided
- generating snippets from registry or release metadata
- distinguishing CLI installation from Epic installation
- minimizing the number of decisions before a user can try the product

Recommended install model:

- detail pages should use a single bootstrap command that accepts the Epic path
  directly and then runs an interactive guided install flow
- the installer should detect or install the CLI as needed, then continue with
  the requested Epic
- direct `epics install <repo>` should remain available in docs and manual
  reference for advanced users and automation, but does not need to be a visible
  primary path on Epic detail pages
- when the user selects Claude, the resulting install should materialize under
  `.claude/skills/<slug>/` in the current project

Recommended bootstrap shape:

- Unix-like systems:
  `curl -fsSL https://epics.sh/install.sh | sh -s -- <epic-path>`
- Windows: PowerShell equivalent that passes `-Epic <epic-path>` to the script
- the installer should ask only the setup questions that materially change the
  result, such as host choice when `--host` is not already specified

This keeps the visible product model simple:

- copy one command from the Epic page
- let the installer handle CLI bootstrap and setup questions

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
- digest-backed review status

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

### 11.1 IA revision based on `skills.sh` benchmark analysis

The original IA proposed 5 top-level nav items and 11 routes. After deep
analysis of `skills.sh` (see Section 3.3–3.7), the IA should be tightened to
reduce navigation friction and merge pages that don't serve unique purposes.

Key principle from `skills.sh`: if a page's content can live inside another
page without hurting discoverability, it should not be a separate nav item.

### 11.2 Revised top-level navigation

- Docs
- CLI

Two links. The homepage IS the directory, so "Directory" is not a nav item.
Compatibility folds into Docs or into per-epic detail pages. Changelog folds
into the CLI section.

### 11.3 Revised route model

Homepage and directory:

- `/` — identity zone + epic directory (the homepage IS the directory)
- `/epics/[slug]` — epic detail page

CLI home (unique to `epics.sh`, not present in `skills.sh`):

- `/cli` — CLI overview, install instructions, downloads
- `/cli/changelog` — release notes and changelog

Docs:

- `/docs` — docs landing
- `/docs/[...slug]` — individual doc pages (getting started, manual,
  compatibility, authoring model, etc.)

### 11.4 Routes removed or merged

| Original route | Disposition |
|---|---|
| `/epics` (directory index) | Merged into `/` — homepage IS the directory |
| `/cli/install` | Merged into `/cli` |
| `/cli/downloads` | Merged into `/cli` |
| `/cli/releases` | Merged into `/cli/changelog` or deferred |
| `/compatibility` | Merged into `/docs/compatibility` or shown per-epic |

### 11.5 Rationale

- `skills.sh` proves that a directory site works best when the homepage is the
  directory itself, not a separate marketing page that links to a directory
- the CLI surface is a legitimate differentiator for `epics.sh` and deserves its
  own nav item, but its sub-pages (install, downloads, releases) can consolidate
  into fewer routes
- compatibility information is most useful in context (on epic detail pages) and
  as a reference doc, not as a standalone top-level page

## 12. Data And Content Dependencies

The web product depends on work outside `apps/web`.

Required upstream inputs:

- Epic listing schema
- maintainer schema
- supported-host schema
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
- a design system that can express supported agents clearly
- strong mobile behavior for install and docs flows

## 14. Success Criteria

The website MVP is successful when:

- a new user can discover an Epic and copy an install command in under 2 minutes
- the site clearly communicates the supported-host contract
- all registry entries have generated detail pages
- CLI download and installation paths are obvious for macOS, Linux, and Windows
- docs and manual are navigable without leaving the site
- the site deploys from CI

## 15. Phasing

### Phase A: PRD and data contracts

- finalize product requirements
- finalize registry and supported-host schemas
- define release metadata inputs for CLI pages

### Phase B: Website MVP foundation

- scaffold `apps/web`
- build landing page, directory, detail page, docs shell, and supported agents page
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
  - install metadata for both the guided bootstrap flow and the direct
    `epics install <repo>` flow
  - digest
  - validation status
  - host-specific caveats

Reason:

- title and summary alone are not sufficient for a trust-oriented directory

### 16.6 Install model

Decision:

- Epic detail pages should default to a bootstrap command that passes the Epic
  path into the installer
- the installer should detect or install the CLI as needed, then ask a short
  series of setup questions
- upfront `--agent` selection should be treated as an advanced path rather than
  the default website flow

Reason:

- this keeps the visible command surface simple while still preserving
  predictable non-interactive and host-specific flows for advanced users

### 16.7 Supported-host contract

Decision:

- every Epic must support every supported host
- supported hosts are global product metadata, not per-Epic compatibility data
- if a host cannot uphold the autonomy contract, it should not be offered at all

Reason:

- this removes noisy compatibility structures and aligns the website with the
  actual user promise of the `epics` CLI

### 16.8 Sample Epic sourcing

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
