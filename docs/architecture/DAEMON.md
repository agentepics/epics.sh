# DAEMON

As of March 7, 2026, the strongest argument for an `epics` daemon is not cron
by itself. It is having one long-running control plane that can own the runtime
parts of the Epic model consistently across host agent CLIs.

OpenClaw Gateway is a useful benchmark because it is not just a scheduler. It
is a supervised, authenticated, long-running process that centralizes control,
health, remote access, background work, and client coordination.

## Short Answer

If `epicsd` only exists to run `cron.d/`, it is probably too much machinery for
V1.

If `epicsd` also handles:

- hook dispatch
- serialized state and log writes
- lifecycle events
- host adapter bridging
- health checks and diagnostics
- background job supervision
- policy-aware runtime decisions

then the daemon starts to justify itself.

That is the main lesson from OpenClaw Gateway.

## What OpenClaw Gateway Does

Based on the current OpenClaw docs, the Gateway acts as a single long-running
control plane for the whole system rather than a narrow helper process.

Key observed properties:

- It is the recommended single process per host for core operations, with
  multiple gateways only for strict isolation or redundancy.
- It owns the main control plane and coordination surface for clients.
- It exposes a stable operator command set for start, stop, restart, install,
  status, logs, and doctor flows.
- It is designed to run under a proper supervisor such as `launchd` or
  `systemd`.
- It includes explicit health checks, repair workflows, auth setup, and service
  drift repair through `openclaw doctor`.
- It multiplexes more than one concern: WebSocket control plane, HTTP surfaces,
  auth, remote access, approvals, stateful sessions, and background process
  management.
- It has a defined protocol with roles, scopes, idempotency expectations, and
  client/device authentication.

This is a materially different model from "a CLI command that occasionally runs
in the background."

## OpenClaw Findings Relevant to epics.sh

### 1. The daemon is valuable because it centralizes ownership

OpenClaw explicitly recommends one Gateway per host and says it is the single
long-running process that owns channel connections and the WebSocket control
plane. That matters because the hardest runtime problems are ownership
problems:

- who owns long-running connections
- who owns scheduled work
- who owns background jobs
- who owns locks and state transitions
- who owns health and restart logic

For `epics.sh`, the same pattern applies. Cron is easier when one process owns:

- the scheduler
- the job ledger
- the lock table
- the state write path
- the hook dispatcher

### 2. Reliability comes from supervision, not from cron syntax

OpenClaw’s runbook emphasizes supervised lifecycle management and provides
`install`, `status`, `restart`, `stop`, and doctor flows rather than treating
the gateway as an ad hoc background process.

That suggests `epicsd` should not be designed as:

- "spawn and hope"
- "background the CLI"
- "run one timer loop in a terminal tab"

It should be designed as:

- an installable user service
- with health checks
- with structured status
- with restart-safe state

If `epics` wants real cron semantics, service management is part of the design,
not an optional extra.

### 3. Health and repair tooling are first-class, not polish

OpenClaw’s `doctor` is notable because it does much more than validate config.
It checks auth readiness, runtime health, service drift, port conflicts, state
layout issues, and repairability.

For `epics.sh`, a daemon would benefit from the same posture:

- `epics doctor`
- `epics daemon status`
- `epics daemon logs`
- `epics daemon install`
- `epics daemon repair`

Without that, a daemon becomes operational debt quickly.

### 4. Security gets harder the moment a daemon becomes a control plane

OpenClaw’s docs point to the same reality: a gateway is useful because it is
central, and risky for the same reason.

Useful security patterns OpenClaw applies:

- loopback-first networking
- auth required by default
- explicit tokens or passwords
- pairing and device identity
- remote-access guidance favoring SSH tunnels or VPN/Tailscale
- fail-closed behavior when auth is missing or the gateway is unavailable

For `epicsd`, this means:

- local-only by default
- no unauthenticated network listener
- no browser UI without a deliberate auth model
- explicit capability scoping for remote clients if remote control is ever added

If `epicsd` remains local IPC only in early versions, that is a feature, not a
limitation.

### 5. A daemon becomes more defensible when it supports multiple runtimes

OpenClaw’s gateway is not only about chat transport. It also supports HTTP
surfaces, operator flows, approvals, remote clients, background exec, and
process management.

The equivalent opportunity for `epicsd` is that one daemon could unify runtime
behavior across:

- Claude Code
- Gemini CLI
- Codex
- OpenCode
- generic shell-based agents

Instead of teaching each host CLI to implement Epic cron, locks, state writes,
and runtime hooks independently, host adapters could become thin clients over a
local daemon.

That is probably the biggest gain.

### 6. Background process management is adjacent to cron

OpenClaw explicitly documents background exec/process handling, including memory
of running jobs, polling, log retrieval, timeouts, and child-process bridging.

That matters because real cron support quickly turns into job-runner support:

- run this job now
- show me active jobs
- fetch job logs
- kill or retry a stuck job
- record exit status
- notify when work completes

If `epicsd` exists, it should probably manage scheduled and ad hoc background
jobs through one runtime model rather than treating cron as a special case.

### 7. The daemon also needs an ingress routing model

Another useful benchmark comes from `claude-to-im-skill`.

Its most transferable idea is not the IM bridge itself. It is the channel
selection strategy: a stable external channel binding on one side, and a more
fragile host-runtime resume pointer on the other.

In that system:

- the bridge keeps a stable mapping from `channelType:chatId` to an internal
  session record
- the host runtime keeps its own opaque session/thread identifier
- the two are linked, but deliberately not collapsed into one ID

That separation is more general than chat.

For `epicsd`, it suggests a broader daemon responsibility:

- own ingress routing
- own stable bindings
- own backend resume pointers
- recover cleanly when a host-side session breaks

Crucially, that ingress routing should be cross-adapter.

If Claude Code, Codex, Gemini CLI, and OpenCode are all installed on the same
machine and connected to `epicsd`, the ingress route should not be bound
directly to only one host CLI. It should be bound first to a daemon-owned
target, while the daemon separately manages which adapter currently executes the
work.

This is relevant to many future capacities:

- IM chats
- webhooks
- dashboard sessions
- cron jobs
- email threads
- MCP clients
- slash-command invocations
- remote API consumers if those ever exist

## What epics.sh Would Gain From a Broader Daemon

If `epicsd` handles more than cron, `epics.sh` gains six concrete things.

### 1. A portable Epic runtime across host CLIs

Today each host CLI has different hooks, extension APIs, and service models.
That makes direct host-specific runtime behavior uneven.

A daemon reduces that problem:

- host CLIs trigger or query `epicsd`
- `epicsd` owns the runtime semantics
- runtime support becomes more consistent even when host adapters differ

This is especially valuable for Codex, where native hook support is still much
weaker than Claude or Gemini.

### 2. Correct cron semantics

Cron support needs more than parsing schedules.

It needs:

- persisted schedules
- missed-run policy
- overlap control
- idempotency keys
- locks
- retries or explicit no-retry rules
- structured run history

A daemon can own these correctly. A plain CLI command cannot do that well by
itself.

### 3. Serialized writes to Epic state

The Epic model has state, plans, logs, roadmap updates, and hook-triggered
changes. Once multiple agents or jobs touch the same Epic, serialized writes
become important.

A daemon can provide:

- per-epic locks
- atomic writes
- monotonic event sequencing
- one canonical append path for logs

That is much cleaner than hoping each host CLI behaves politely.

### 4. A unified event bus

The daemon could normalize events into one internal model:

- cron fired
- plan empty
- milestone complete
- became blocked
- session started
- session stopped
- state changed

Then host adapters only need to translate native events into daemon events.

That is cleaner than encoding Epic runtime logic separately in Claude hooks,
Gemini hooks, OpenCode plugins, and Codex notifications.

### 5. Better diagnostics and operability

Once there is a daemon, `epics` can support real operations workflows:

- inspect active Epics
- inspect scheduled jobs
- inspect last successful run
- inspect failures
- inspect unsupported runtime features
- inspect host adapter connectivity

This is the difference between "some config files exist" and "the runtime is
observable."

### 6. Room for later features

A broader daemon could later support:

- local API for the website or desktop UI
- registry sync and update checks
- policy enforcement hooks
- local approval queues
- notifications
- multi-workspace or multi-user isolation

Those are not V1 requirements, but they become possible once the runtime has a
real control plane.

### 7. Stable routing across many ingress channels

The `claude-to-im-skill` pattern suggests an additional gain that deserves to
be explicit: the daemon can become the stable routing authority across many
ingress types.

The generalized model looks like this:

1. **Ingress identity**
   A stable external identifier such as:
   - `telegram:<chat_id>`
   - `discord:<channel_id>`
   - `webhook:<endpoint_id>:<tenant>`
   - `cron:<epic_id>:<job_name>`
   - `dashboard:<user_id>:<workspace>`
   - `mcp:<client_id>:<session>`

2. **Daemon-side binding**
   A durable local record that answers:
   - which Epic or capability this ingress maps to
   - which workspace or working directory it uses
   - which mode, model, or policy profile it prefers
   - any route-local metadata and permissions

3. **Adapter-side execution binding**
   A selected host adapter plus its current opaque resume handle, such as:
   - `adapter=claude`, `session_id=...`
   - `adapter=codex`, `thread_id=...`
   - `adapter=gemini`, `session_id=...`
   - `adapter=opencode`, `agent_session=...`

4. **Executor-side context pointer**
   A host-specific opaque resume handle such as:
   - Claude session ID
   - Codex thread or resume token
   - Gemini session/thread identifier
   - OpenCode session or agent-local context ID

The key design point is that the daemon-side binding is the durable identity.
The adapter selection and host-specific context pointer are replaceable.

That creates a powerful failure model:

- an IM chat does not lose its binding if the Claude session breaks
- a webhook route does not disappear because a Codex thread expired
- a cron job keeps targeting the same Epic even if the underlying host session
  is reset
- a dashboard view can stay attached to the same Epic while the daemon rotates
  or recreates executor state underneath it
- if Claude is unavailable, the daemon can rebind execution to Gemini or Codex
  without changing the logical route

This is probably the cleanest way to make many daemon capabilities feel
consistent without forcing all hosts to expose the same session primitives.

## Generalizing The Channel Selection Strategy

The `claude-to-im-skill` approach can be generalized into a cross-adapter
routing model for
`epicsd`.

At minimum, it should be treated as a four-layer routing model:

1. ingress binding
2. daemon object binding
3. adapter selection
4. executor binding

If only one host is installed, layers 3 and 4 may look trivial. If several
hosts are installed, they become essential.

### Layer 1: ingress binding

Map every inbound interaction to a stable route key.

Examples:

- IM chat
- webhook endpoint
- cron trigger
- UI session
- MCP client
- CLI command source

The route key should be durable and host-agnostic.

### Layer 2: daemon object binding

Map that route key to the daemon’s own durable object, which might be:

- an Epic instance
- an installed capability epic
- a workspace-scoped controller
- a multi-epic router profile

This binding should hold local policy and metadata:

- preferred workspace
- preferred host runtime
- mode
- model hints
- auth scopes
- rate limits
- allowed actions

### Layer 3: adapter selection

Select which installed host adapter should execute the work for this route.

That selection should be policy-driven, not implicit.

Examples:

- prefer Claude for interactive chats
- prefer Codex for batch execution
- prefer Gemini when a route requires hook semantics Codex cannot provide
- prefer OpenCode when a plugin-backed workflow is available

The selection policy could use:

- route type
- Epic type
- required capability
- user preference
- cost or quota status
- health status
- fallback order

This is the missing generalization if several CLIs are installed at once.

### Layer 4: executor binding

Map the daemon object to the current execution context in the chosen host.

Examples:

- current Claude session ID
- current Codex resume target
- current Gemini runtime context
- current OpenCode agent/session

This binding is intentionally ephemeral and replaceable.

If it breaks:

- clear only the adapter/executor binding
- keep the ingress and daemon object bindings
- pick the same adapter or another eligible adapter
- start a fresh host session underneath

That is the architectural lesson from the two-sided binding strategy.

## What This Unlocks For epicsd

Once generalized, the same routing model can support many daemon capacities.

### IM chat routing

- one chat stays bound to one Epic or workspace
- `/new` can create a fresh executor context
- `/bind` can re-point the chat to another Epic or existing daemon object

### Webhook routing

- each webhook endpoint or tenant key routes to a durable daemon binding
- retries and deduplication happen at the daemon layer
- the backend executor session can be recreated without changing the route
- adapter selection can vary by endpoint policy or required capability

### Cron routing

- each scheduled job can route to a stable Epic binding
- the scheduler triggers the route, not a raw shell command
- executor state can be reused or recreated according to policy
- execution can be routed to the best installed adapter for unattended work

### Dashboard routing

- UI tabs or users can attach to stable daemon objects
- the daemon can stream status/logs even when no host session is currently
  active
- "resume" becomes an operation on the daemon object, not just on the host
- the dashboard can surface which adapter currently owns execution and why

### MCP and local API routing

- each connected client gets a route identity
- permissions and rate limits can be applied before work reaches the host agent
- daemon-side bindings keep the logical target stable across host restarts
- clients can request a preferred adapter without owning final adapter choice

### Multi-host failover or host selection

This is especially important for `epics.sh`.

A route could target one logical Epic while the daemon chooses among:

- Claude when hooks are needed
- Codex when structured non-interactive execution is preferred
- Gemini when a richer hook or extension surface is available
- OpenCode when a plugin-backed agent flow is desired

The ingress route does not need to change. Only the adapter/executor binding
changes.

That would let `epicsd` express policies like:

- prefer Claude for interactive work
- prefer Codex for `exec` jobs
- fall back to Gemini if Claude is unavailable

without changing the logical route or Epic target.

That is the core requirement for cross-adapter ingress routing.

## Recommended epicsd Data Model

If `epicsd` adopts this pattern, it likely needs several distinct stores:

- `routes/` or `bindings/`
  Stable ingress identity -> daemon object mapping
- `objects/`
  Durable daemon-owned records for Epics, capabilities, or workspace routers
- `adapters/`
  Installed adapter inventory, health, capabilities, and selection policy
- `executors/`
  Current per-adapter host-runtime resume pointers and health
- `runs/`
  Job and invocation ledger
- `locks/`
  Per-object or per-Epic serialization state

This is more robust than trying to treat one session ID as the entire identity
of the system.

## Recommended Operator Actions

The `claude-to-im-skill` override pattern also generalizes well.

Equivalent daemon actions would be:

- `epics route new <route>`
  Create a fresh daemon object or fresh executor context for a route
- `epics route bind <route> <target>`
  Re-point an ingress route to a different Epic, workspace, or daemon object
- `epics route inspect <route>`
  Show ingress binding, daemon object, adapter choice, and executor binding
  separately
- `epics route reset-executor <route>`
  Clear only the host-side resume pointer
- `epics route update <route>`
  Change metadata like workspace, mode, host, or model preference
- `epics route select-adapter <route> <adapter>`
  Pin or override adapter selection for a route
- `epics adapter list`
  Show installed adapters, capabilities, and health
- `epics adapter doctor`
  Validate adapter connectivity and runtime readiness

This would make daemon routing observable and debuggable instead of implicit.

## What epics.sh Would Lose

The costs are real.

### 1. More security surface

A daemon with sockets, APIs, or remote control becomes a sensitive local
service. OpenClaw is a good reminder that control-plane security is part of the
product, not a post-launch hardening task.

### 2. More operational burden

Now `epics.sh` needs:

- service install/uninstall flows
- logs
- lock cleanup
- crash recovery
- upgrades
- migrations
- health checks

### 3. More product scope

Once a daemon exists, users will expect it to be dependable. That raises the
bar on testing, support, and backward compatibility.

### 4. Risk of over-centralization

If too much logic moves into the daemon too early, the simple CLI workflows
become harder to understand and harder to debug.

## Recommended Direction

The best path is probably:

### Phase A: no mandatory daemon

Ship the CLI first with:

- validation
- install
- resume/export
- host setup
- state/plan/log helpers
- `epics cron validate`

This keeps V1 simple.

### Phase B: optional local daemon

Add `epicsd` as an optional runtime sidecar for users who want:

- real cron execution
- hook dispatch
- background jobs
- runtime status
- stronger multi-agent coordination

This keeps the base CLI usable while allowing a stronger runtime profile.

### Phase C: daemon-backed host adapters

Host adapters should then prefer:

- send event to `epicsd`
- request context from `epicsd`
- ask `epicsd` to run hooks or jobs

Instead of embedding deep runtime semantics inside each host integration.

## Recommended epicsd Scope

If `epicsd` is built, the smallest scope that justifies it is:

- local-only daemon
- ingress router with stable route bindings
- cross-adapter selection layer
- per-epic scheduler for `cron.d/`
- per-epic locking
- hook dispatch engine
- executor binding store for host session pointers
- run ledger with status and logs
- health/status endpoint over local IPC
- service install for macOS and Linux
- `epics doctor` support for daemon health

That is enough to be useful without turning into a full OpenClaw-like platform.

## Recommended Non-Scope For Early epicsd

Avoid these initially:

- remote network access
- browser control plane
- user-facing multi-tenant auth
- distributed execution across machines
- full generic plugin marketplace
- attempting to replace host CLIs

## Bottom Line

OpenClaw Gateway suggests that a daemon is worth building when it becomes the
runtime authority for several adjacent concerns, not when it is only a cron
wrapper.

For `epics.sh`, the strongest case for `epicsd` is:

- portable runtime behavior across inconsistent host CLIs
- correct scheduling and background execution
- serialized writes and event coordination
- observable health and repair tooling

The weakest case is:

- "we need something that runs cron expressions"

That narrower problem can be solved more cheaply. The broader control-plane
problem is where a daemon starts to pay for itself.

## Sources

- OpenClaw Gateway runbook:
  https://docs.openclaw.ai/gateway
- OpenClaw Gateway protocol:
  https://docs.openclaw.ai/gateway/protocol
- OpenClaw network model:
  https://docs.openclaw.ai/gateway/network-model
- OpenClaw doctor:
  https://docs.openclaw.ai/gateway/doctor
- OpenClaw configuration reference:
  https://docs.openclaw.ai/gateway/configuration-reference
- OpenClaw background exec/process docs:
  https://docs.openclaw.ai/gateway/background-process
- OpenClaw remote access:
  https://docs.openclaw.ai/gateway/remote
- Benchmark repo:
  https://github.com/op7418/claude-to-im-skill
