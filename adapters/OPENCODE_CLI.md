# OpenCode CLI Adapter

## Summary

OpenCode has the broadest officially documented programmatic extension surface
of the four host CLIs in this study.

The strongest official methods are:

- `AGENTS.md` rules
- `opencode.json`
- custom commands
- plugins
- custom agents and subagents
- Agent Skills
- MCP
- server modes such as `web`, `serve`, and `acp`

For `epics.sh`, OpenCode is not the easiest host to support first, but it may
be the most flexible long term.

## Official Integration Methods

### 1. `AGENTS.md` rules

OpenCode officially supports `AGENTS.md` for custom instructions and documents:

- project rules
- global rules in `~/.config/opencode/AGENTS.md`
- Claude compatibility fallbacks to `CLAUDE.md`
- precedence across local and global rule files

This is a strong baseline for `epics.sh` because it supports:

- generic agent-compatible instructions
- migration paths from Claude-oriented workspaces
- portable Epic guidance without requiring a plugin

OpenCode also documents support for external rule references and modular rule
loading patterns.

### 2. `opencode.json`

OpenCode officially uses `opencode.json` or JSONC config with:

- global config
- project config
- custom config path
- custom config directory

This is the main adapter surface for:

- tools
- MCP
- plugins
- agent configuration
- server configuration

For `epics.sh`, it is the natural place for generated adapter config.

### 3. Custom commands

OpenCode officially supports custom commands in:

- `.opencode/commands/`
- config-defined command entries

These commands can specify:

- template prompt
- description
- agent
- model

This is a strong wrapper layer for `epics.sh` operator actions such as:

- `/epics-status`
- `/epics-resume`
- `/epics-validate`

As with the other hosts, the command file should remain a thin shim over the
`epics` CLI.

### 4. Plugins

OpenCode officially supports plugins from:

- `.opencode/plugins/`
- `~/.config/opencode/plugins/`
- npm packages referenced from config

This is its most distinctive integration surface.

The plugin model supports:

- JavaScript/TypeScript implementation
- event subscriptions
- access to project and directory context
- an SDK client for interacting with OpenCode
- shell execution via Bun

This means OpenCode is the best target if `epics.sh` ever needs rich host-side
logic without forcing everything through shell hooks.

### 5. Agents and subagents

OpenCode officially supports:

- built-in primary agents
- built-in subagents
- custom agents
- `opencode agent create`
- agent-specific tool permissions
- `@` mention invocation

This is a strong fit for Epic helper roles such as:

- planning-only Epic agent
- read-only Epic explorer
- review or audit agents

### 6. Agent Skills

OpenCode officially supports `SKILL.md`-based Agent Skills and discovers them
from:

- `.opencode/skills/`
- `~/.config/opencode/skills/`
- `.claude/skills/`
- `~/.claude/skills/`
- `.agents/skills/`
- `~/.agents/skills/`

This is highly relevant to `epics.sh`.

It means OpenCode is already compatible with the broader agent-skill ecosystem
that Epics build on.

This should make OpenCode one of the most natural homes for Epic installation.

### 7. MCP

OpenCode officially supports local and remote MCP servers configured in
`opencode.json`.

It also documents:

- OAuth handling for remote MCP
- per-agent MCP enablement
- tool-level enable/disable control
- debugging and auth flows

This is an excellent future integration path for `epics.sh`, especially if
`epics` or `epicsd` becomes an MCP server.

### 8. Server modes and protocols

OpenCode officially documents several server-facing modes, including:

- `opencode web`
- `opencode serve`
- `opencode acp`
- attach to a running server instance

This is a meaningful difference from the other hosts. It means OpenCode already
expects more headless, service-like, and protocol-driven deployments.

That aligns well with an eventual Epic daemon or control plane.

## Other Possible OpenCode Integration Methods

### Method A: `AGENTS.md` + skills only

Good for:

- lowest-friction installation
- host-neutral packaging
- immediate compatibility

This is the best first OpenCode adapter cut.

### Method B: `opencode.json` + commands + agents

Good for:

- richer operator UX
- project-shared setup
- clearer role separation between planning and execution

### Method C: plugin-backed Epic adapter

Good for:

- richer event handling
- host-side logic
- integrating with server modes or external services

This is powerful, but more expensive to maintain than a shell-driven adapter.

### Method D: MCP or daemon-backed runtime

Good for:

- structured Epic tools
- runtime state inspection
- deeper lifecycle support

Long term, this is probably the cleanest advanced OpenCode path.

## Recommended Adapter Strategy For epics.sh

Phase order:

1. `AGENTS.md` and skills compatibility
2. `epics host setup opencode` generating `opencode.json` and commands
3. optional custom agents for Epic roles
4. optional plugin-backed adapter
5. optional MCP or daemon-backed advanced runtime

## Sources

- OpenCode rules:
  https://opencode.ai/docs/rules
- OpenCode config:
  https://opencode.ai/docs/config/
- OpenCode commands:
  https://opencode.ai/docs/commands/
- OpenCode plugins:
  https://opencode.ai/docs/plugins/
- OpenCode agents:
  https://opencode.ai/docs/agents/
- OpenCode skills:
  https://opencode.ai/docs/skills
- OpenCode MCP servers:
  https://opencode.ai/docs/mcp-servers/
- OpenCode CLI:
  https://opencode.ai/docs/cli/
- OpenCode ACP:
  https://opencode.ai/docs/acp/
