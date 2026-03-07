# SPEC_EXTENSION

This document records possible future Epic spec extensions that only become
plausible if `epicsd` reaches the Phase C model described in
[DAEMON.md](./DAEMON.md):

- optional local daemon
- daemon-backed host adapters
- cross-adapter ingress routing
- adapter selection as a daemon concern
- executor/session binding owned below the daemon object layer

## Critical Constraint

These are **conditional runtime extensions**, not current Epic requirements.

They should be understood as:

- possible future extensions
- optional for executors that implement a stronger runtime
- not required for a valid Epic package
- not part of the portable minimum Epic definition unless explicitly promoted
  later

The core Epic model should remain file-portable.

That means:

- `SKILL.md`
- `EPIC.md`
- `plans/`
- `state.json` or `state/`
- `log/`
- `ROADMAP.md`
- `DECISIONS.md`
- `hooks/`
- `cron.d/`
- `policy.yml`

must continue to define the portable baseline.

Anything in this document should be treated as an optional runtime layer that a
more capable executor may implement.

## Why These Extensions Only Make Sense After Phase C

Before Phase C, `epics.sh` is mostly:

- a CLI
- host setup helpers
- adapter-specific integration glue

At that stage, the system does not yet have a strong enough runtime authority
to standardize cross-adapter concerns such as:

- stable ingress routes
- host selection policy
- failover between installed CLIs
- daemon-owned session routing
- normalized run ledgers

After Phase C, the daemon becomes a real cross-adapter runtime control plane.
That is the point where these concepts become coherent enough to discuss as
spec-level extensions.

## Candidate Conditional Extensions

### 1. Route bindings

Potential extension:

- a standard optional runtime concept for stable ingress identities

Examples:

- IM chat route
- webhook route
- dashboard route
- cron route
- MCP client route

Purpose:

- let a durable external route map to a daemon-owned Epic target without being
  tied directly to one host session

Conditionality:

- only meaningful when a daemon or equivalent executor owns routing
- not required for ordinary Epic usage

### 2. Adapter selection policy

Potential extension:

- declarative policy for selecting among installed host adapters such as Claude,
  Codex, Gemini, or OpenCode

Examples:

- prefer Claude for interactive work
- prefer Codex for batch execution
- prefer Gemini when a hook-capable executor is needed
- fall back to OpenCode for plugin-backed workflows

Purpose:

- make cross-adapter routing an explicit runtime concern

Conditionality:

- only meaningful when multiple host adapters are installed and connected
- irrelevant for single-host or CLI-only environments

### 3. Executor state metadata

Potential extension:

- a standard optional runtime representation of active adapter ownership and
  host-side resume pointers

Examples:

- current adapter
- session/thread identifier
- executor health
- last successful resume
- adapter failover state

Purpose:

- give runtime-aware tooling a consistent way to inspect live execution state

Conditionality:

- not part of the portable Epic package
- belongs to executor/runtime metadata, not the core spec

### 4. Run ledger semantics

Potential extension:

- normalized runtime records for invocations and their outcomes

Examples:

- run IDs
- trigger source
- adapter used
- start/end timestamps
- idempotency key
- retry count
- exit result
- log linkage

Purpose:

- create consistent observability across cron, hooks, webhooks, and manual
  invocations

Conditionality:

- only needed once the runtime can own and track many invocation types
- should remain optional executor metadata, not a required core file

### 5. Trigger routing

Potential extension:

- a broader trigger model than today’s local `hooks/` and `cron.d/`

Possible trigger classes:

- webhook-received
- message-received
- dashboard-action
- mcp-call
- schedule-fired
- session-started
- session-stopped

Purpose:

- allow richer runtime event handling through a daemon-owned event bus

Conditionality:

- should only exist if the runtime can actually observe and route such events
- must not be implied by plain `SKILL.md` guidance alone

### 6. Locking and concurrency policy

Potential extension:

- declarative overlap and serialization rules

Examples:

- single-flight
- queue
- replace
- parallel
- per-route lock
- per-epic lock

Purpose:

- govern concurrent runtime behavior in a standardized way

Conditionality:

- only useful when the executor can actually enforce it
- not appropriate as a portable core requirement

### 7. Ingress auth and trust scopes

Potential extension:

- optional runtime policy around who or what may trigger a route

Examples:

- allowed webhook signer
- allowed IM user or channel
- allowed dashboard role
- allowed MCP client scope

Purpose:

- allow a daemon-backed runtime to describe trigger permissions formally

Conditionality:

- only relevant when an Epic is exposed through ingress routes
- should remain runtime policy, not a baseline Epic file requirement

### 8. Capability declarations

Potential extension:

- machine-readable declarations of runtime features an Epic expects or can use

Examples:

- requires cron execution
- supports webhook ingress
- supports IM ingress
- supports daemon-managed routing
- supports cross-adapter failover
- supports approval mediation

Purpose:

- help adapters and runtimes surface compatibility accurately

Conditionality:

- should be optional capability metadata
- must not invalidate otherwise valid Epics that do not use those features

### 9. Installed capability Epic profile

Potential extension:

- a clearer standard profile for long-lived installed capability Epics

Examples:

- memory service
- monitoring service
- inbound communications bridge
- sync or reporting service

Purpose:

- distinguish manually resumed project Epics from daemon-managed installed
  capabilities

Conditionality:

- only useful once daemon-managed routing and lifecycle exist
- should remain a profile/archetype extension, not a new minimum requirement

### 10. Portability tiers

Potential extension:

- explicit classification between:
  - portable core Epic files
  - optional runtime extensions
  - host-specific adapter metadata

Purpose:

- prevent runtime-heavy features from being mistaken for baseline Epic
  requirements

Conditionality:

- mainly useful once the ecosystem includes both simple and daemon-backed
  executors

## Recommended Guardrails

If any of these are ever proposed formally, the spec work should keep these
guardrails:

### Guardrail 1: core Epics remain valid without daemon support

A valid Epic must not require:

- a daemon
- network listeners
- cross-adapter routing
- live session binding

unless the spec is explicitly split into core and optional runtime profiles.

### Guardrail 2: executor metadata stays separate from portable files

Live adapter state, session pointers, locks, and route bindings should stay in
runtime-managed storage, not become required portable files inside the Epic.

### Guardrail 3: unsupported runtime features must be surfaced explicitly

If an Epic declares optional runtime capabilities, an executor must either:

- honor them
- surface that it cannot

It must not silently downgrade them into advisory prose.

### Guardrail 4: host-specific details should not leak into the portable spec

The spec should not standardize raw Claude, Codex, Gemini, or OpenCode session
fields as portable requirements.

Portable runtime concepts should stay generic:

- route
- adapter
- executor
- capability
- trigger
- run

### Guardrail 5: multi-host routing remains optional

Cross-adapter ingress routing should be a high-capability runtime feature, not
the assumption every Epic is built around.

## Practical Interpretation

The safe interpretation is:

- before Phase C, these concepts are design ideas
- after Phase C, they become plausible runtime-extension candidates
- even then, they should begin as optional executor extensions, not core spec
  requirements

That is the main point of this document.

## Bottom Line

If `epicsd` never reaches a true cross-adapter Phase C runtime, most of the
extensions listed here should remain out of the Epic spec.

If `epicsd` does reach that level, the Epic ecosystem may reasonably grow an
optional runtime-extension layer covering:

- route bindings
- adapter selection
- executor metadata
- run ledgers
- richer trigger routing
- locking policy
- ingress trust scopes
- capability declarations
- installed capability profiles
- portability tiers

But those should remain explicitly conditional and optional unless the standard
is deliberately revised to define a stronger runtime profile.
