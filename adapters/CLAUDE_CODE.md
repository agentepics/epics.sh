# Claude Code Adapter

## Summary

Claude Code currently has the richest official integration surface of the four
host CLIs relevant to `epics.sh`.

The most useful official adapter methods are:

- `CLAUDE.md` memory files
- hierarchical `settings.json`
- hooks
- custom slash commands
- subagents
- MCP client and server modes

There is also an empirical plugin distribution pattern in the wild, shown by
`claudeclaw`, but the most durable official surfaces are the ones above.

## Official Integration Methods

### 1. `CLAUDE.md` memory files

Claude Code officially supports persistent instruction files through
`CLAUDE.md`. Anthropic documents:

- user memory in `~/.claude/CLAUDE.md`
- project memory via `CLAUDE.md`
- recursive loading up the directory tree
- `/memory` for inspection and editing

This is the cleanest place to inject:

- Epic-aware working conventions
- "how to resume this workspace" guidance
- preferred `epics` CLI commands
- adapter-specific recovery instructions

For `epics.sh`, this is the best baseline context surface in Claude Code.

### 2. Hierarchical settings via `.claude/settings.json`

Claude Code officially supports:

- `~/.claude/settings.json`
- `.claude/settings.json`
- `.claude/settings.local.json`
- enterprise-managed policy

This is the main official surface for:

- hooks
- permissions
- environment variables
- project-shared integration config

For `epics.sh`, this is the right place for generated project-scoped adapter
configuration, especially when installing shared Epic workflows into a repo.

### 3. Hooks

Claude Code’s hook system is one of its strongest integration features.

Officially documented events include:

- `PreToolUse`
- `PostToolUse`
- `Notification`
- `UserPromptSubmit`
- `Stop`
- `SubagentStop`
- `PreCompact`
- `SessionStart`
- `SessionEnd`

Anthropic also documents:

- matcher-based hook routing
- shell command hooks
- structured JSON output
- blocking behavior
- context injection for some events
- project hook trust and review flow

This makes Claude Code the strongest first target for Epic lifecycle helpers
such as:

- session-start resume context
- stop-time session logging
- write-time validation around Epic files
- explicit warnings when runtime features are unsupported

### 4. Custom slash commands

Claude Code officially supports custom slash commands stored in:

- `.claude/commands/`
- `~/.claude/commands/`

They are Markdown files with frontmatter, and support:

- descriptions
- allowed tool scopes
- arguments
- specific models
- file references with `@...`

This is a strong UX layer for `epics.sh` because it can expose a friendlier
operator surface without duplicating core logic.

Examples:

- `/epics:status`
- `/epics:resume`
- `/epics:doctor`
- `/epics:plan-next`

The important design rule is that these commands should delegate quickly to the
`epics` CLI rather than encode runtime logic themselves.

### 5. Subagents

Claude Code officially supports custom subagents in:

- `~/.claude/agents/`
- `.claude/agents/`

Subagents can define:

- purpose
- description
- tool access
- custom prompts

This creates a plausible Epic integration method for specialized workflows,
such as:

- an Epic planner
- an Epic validator
- an Epic historian/log summarizer

For `epics.sh`, this is useful, but secondary. Subagents are a good UX layer,
not the canonical runtime.

### 6. MCP

Anthropic officially supports both:

- Claude Code as an MCP client
- Claude Code as an MCP server via `claude mcp serve`

It also supports project-scoped MCP configuration through `.mcp.json`.

This opens two different adapter directions for `epics.sh`:

- `epics` or `epicsd` as an MCP server exposing Epic-aware tools
- Claude Code project setup that adds shared Epic MCP services to `.mcp.json`

Long term, this is probably the cleanest advanced integration path if `epics.sh`
needs richer structured tool access than slash commands and hooks alone.

## Empirical Benchmark Patterns

### ClaudeClaw

`claudeclaw` demonstrates a strong Claude-native pattern:

- plugin packaging
- slash-command UX
- project-local daemon state
- recurring jobs
- local dashboard

Useful takeaways:

- Claude users respond well to native command-driven UX
- project-local integration feels natural for repo-specific automation

Main risk:

- this approach is too Claude-specific to be the canonical `epics.sh` model

### Claude-to-IM Skill

`claude-to-im-skill` is less Claude-native in packaging, but architecturally
better for `epics.sh` because it separates:

- host wrapper
- daemon/process lifecycle
- provider/runtime logic

That is the better long-term pattern even for Claude.

## Other Possible Claude Code Integration Methods

Beyond the benchmark projects, the practical methods are:

### Method A: `CLAUDE.md` + `epics` CLI only

Good for:

- lowest-friction install
- no background runtime
- repo-local Epic workflows

This is the safest first integration cut.

### Method B: `CLAUDE.md` + slash commands + hooks

Good for:

- first-class interactive UX
- reliable session lifecycle integration
- consistent validation and logging

This is probably the best V1 Claude adapter target.

### Method C: MCP-backed adapter

Good for:

- richer structured tooling
- future UI or daemon integration
- stronger machine-readable interactions

This is stronger architecturally, but likely a later-phase adapter.

### Method D: daemon-backed Claude adapter

Good for:

- `cron.d/`
- hook dispatch normalization
- serialized state writes
- run ledgers and diagnostics

This should be optional, not required for the first Claude release.

## Recommended Adapter Strategy For epics.sh

Phase order:

1. `CLAUDE.md` + `epics host setup claude`
2. generated slash commands
3. generated hooks for resume/logging/validation
4. optional MCP or daemon-backed advanced runtime

What Claude Code should become for `epics.sh`:

- the strongest interactive adapter
- not the definition of Epic semantics

## Sources

- Anthropic Claude Code settings:
  https://docs.anthropic.com/en/docs/claude-code/settings
- Anthropic Claude Code slash commands:
  https://docs.anthropic.com/en/docs/claude-code/slash-commands
- Anthropic Claude Code subagents:
  https://docs.anthropic.com/en/docs/claude-code/sub-agents
- Anthropic Claude Code hooks:
  https://docs.anthropic.com/en/docs/claude-code/hooks
- Anthropic Claude Code memory:
  https://docs.anthropic.com/en/docs/claude-code/memory
- Anthropic Claude Code MCP:
  https://docs.anthropic.com/en/docs/claude-code/mcp
- Benchmark repo:
  https://github.com/moazbuilds/claudeclaw
- Benchmark repo:
  https://github.com/op7418/claude-to-im-skill
