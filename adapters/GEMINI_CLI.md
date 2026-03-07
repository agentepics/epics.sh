# Gemini CLI Adapter

## Summary

Gemini CLI has a broad official extension surface and is the closest current
peer to Claude Code for hook-based integration.

The strongest official methods are:

- `GEMINI.md`
- custom commands
- `settings.json`
- hooks
- extensions
- Agent Skills
- sub-agents
- MCP

For `epics.sh`, Gemini should be treated as a first-class adapter target, not a
fallback.

## Official Integration Methods

### 1. `GEMINI.md`

Gemini CLI officially supports `GEMINI.md` context files, searched from the
current directory upward to the project root, plus global memory under
`~/.gemini/GEMINI.md`.

Gemini also documents:

- `/memory`
- modular memory imports
- `save_memory`

For `epics.sh`, this is the best baseline instruction surface for:

- Epic operating rules
- repo-local resume guidance
- install-time guidance for adapter-specific commands

This is the closest Gemini equivalent to Claude’s `CLAUDE.md`.

### 2. Custom commands

Gemini CLI officially supports custom commands in:

- `~/.gemini/commands/`
- `<project>/.gemini/commands/`

These are TOML-defined and support:

- descriptions
- namespacing
- argument substitution
- shell command interpolation via `!{...}`
- file injection via `@{...}`

This gives `epics.sh` a strong UX wrapper layer for commands like:

- `/epics:resume`
- `/epics:status`
- `/epics:doctor`
- `/epics:validate`

As with Claude, these should delegate to `epics` rather than implement runtime
logic themselves.

### 3. `settings.json`

Gemini CLI officially documents settings at:

- `.gemini/settings.json`
- `~/.gemini/settings.json`
- `/etc/gemini-cli/settings.json`

This is the main surface for:

- hooks
- project-scoped adapter config
- shared install behavior

It is the Gemini equivalent of Claude’s hierarchical settings model.

### 4. Hooks

Gemini CLI officially supports synchronous hooks configured in `settings.json`.

Documented traits include:

- tool event hooks such as `BeforeTool` and `AfterTool`
- lifecycle hooks
- regex matchers for tool events
- extension-bundled hooks
- project hook trust/fingerprinting
- environment variables including `GEMINI_PROJECT_DIR`
- a `CLAUDE_PROJECT_DIR` compatibility alias

This is highly relevant to `epics.sh` because it enables:

- resume context loading
- write-time validation
- stop-time logging
- lifecycle-triggered Epic helpers

Gemini is therefore a credible first-wave host for Epic hook integration.

### 5. Extensions

Gemini CLI officially supports extensions as first-class packaging units.
Extensions can bundle:

- custom commands
- hooks
- sub-agents
- Agent Skills
- MCP configuration

This is the strongest official packaging surface Gemini offers beyond plain
workspace files.

For `epics.sh`, there are two plausible strategies:

- keep Epics host-neutral and generate Gemini workspace files
- later publish an `epics` Gemini extension that bundles helper commands,
  skills, and setup logic

The first should come first. The second is a later UX layer.

### 6. Agent Skills

Gemini CLI officially supports Agent Skills, based on the Agent Skills open
standard. It supports:

- workspace skills
- user skills
- extension-bundled skills
- `gemini skills` management commands
- `.agents/skills/` as a compatibility alias

This is directly relevant to `epics.sh`.

It means Gemini is already aligned with the packaging direction most useful for
portable Epic capabilities.

### 7. Sub-agents

Gemini extensions can bundle sub-agents in `agents/`.

This makes Gemini a good fit for specialized Epic helpers such as:

- Epic planner
- Epic validator
- Epic migration assistant

As with Claude, sub-agents should be treated as useful UX layers, not the core
runtime.

### 8. MCP

Gemini CLI supports MCP and documents commands like `gemini mcp add`.

This provides a future path for:

- Epic-aware tool servers
- daemon-backed structured integration
- richer host-neutral tooling than commands and hooks alone

## Other Possible Gemini CLI Integration Methods

### Method A: `GEMINI.md` + custom commands

Good for:

- low-friction install
- portable instructions
- easy project-level onboarding

### Method B: `GEMINI.md` + hooks + settings

Good for:

- first-class Epic lifecycle integration
- validation and logging
- reliable workspace sharing

This is likely the best V1 Gemini adapter cut.

### Method C: extension-backed Epic helper package

Good for:

- polished distribution
- bundled commands, hooks, skills, and sub-agents
- richer out-of-the-box UX

This is a strong later-phase option.

### Method D: MCP or daemon-backed runtime

Good for:

- structured state/log operations
- runtime capability reporting
- deeper Epic runtime support

## Recommended Adapter Strategy For epics.sh

Phase order:

1. `GEMINI.md` guidance + `epics host setup gemini`
2. generated commands and settings snippets
3. generated hooks for resume/logging/validation
4. optional Gemini extension
5. optional MCP or daemon-backed runtime integration

## Sources

- Gemini CLI context files:
  https://geminicli.com/docs/cli/gemini-md
- Gemini CLI custom commands:
  https://geminicli.com/docs/cli/custom-commands/
- Gemini CLI settings:
  https://geminicli.com/docs/cli/settings
- Gemini CLI hooks:
  https://geminicli.com/docs/hooks/
- Gemini CLI extensions:
  https://geminicli.com/docs/extensions/
- Gemini CLI extension reference:
  https://geminicli.com/docs/extensions/reference/
- Gemini CLI Agent Skills:
  https://geminicli.com/docs/cli/skills/
- Gemini CLI creating skills:
  https://geminicli.com/docs/cli/creating-skills/
- Gemini CLI CLI reference:
  https://geminicli.com/docs/cli/cli-reference/
