# E2E Epic Testing

This document describes the Claude Code Epic E2E environment in this repo.

## Purpose

The Claude E2E lane verifies that:

- Claude Code itself can run headlessly inside Docker
- `epics install` can install a real Epic into a project-local Claude workspace
- the EPIC-spec `install` trigger is executed during install
- Claude can discover and read the installed Epic from the project

## Auth setup

Claude-backed scenarios require an Anthropic API key in `ANTHROPIC_API_KEY`.
The recommended local secret file is:

```text
.env.e2e
```

That file is gitignored. The E2E runner loads it automatically before scenario
selection.

Example:

```bash
cp .env.e2e.example .env.e2e
# edit .env.e2e and set ANTHROPIC_API_KEY
go run ./e2e/cmd/epics-e2e run --tag claude --keep-artifacts
```

If the key is absent, the selected E2E run now fails during preflight with a
clear error. It does not skip Claude scenarios.

Official references used for this setup:

- Claude Code settings and environment variables:
  https://docs.anthropic.com/en/docs/claude-code/settings
- Claude Code SDK / npm installation:
  https://docs.claude.com/es/docs/claude-code/sdk

## Docker environment

The Claude runner image lives in `e2e/docker/claude-runner.Dockerfile`.

It contains:

- the locally built `epics` binary
- `git` for remote GitHub Epic fetches
- `python3` for simple script hooks and the backend fixture
- `node` / `npm`
- `@anthropic-ai/claude-code`

The fixture project lives under `e2e/fixtures/claude-web-project/` and provides:

- `backend/app.py`
- `frontend/` Vite React app files

The hook-validation Epic fixtures live under:

- `examples/fixtures/install-hook-epic/` for a real `.sh` install hook
- `examples/fixtures/prompt-install-hook-epic/` for a real `.md` prompt hook

## Scenarios

`claude-hello-world`
- Runs `claude -p "Respond exactly 'Hello world!' and nothing else." --dangerously-skip-permissions --output-format text`
- Requires `ANTHROPIC_API_KEY`
- Asserts exact stdout

`claude-install-remote-epic`
- Installs a real Epic from `https://github.com/agentepics/epics/tree/main/autonomous-coding`
- Verifies `.claude/skills/autonomous-coding/` plus CLI metadata and Claude wrapper files
- Verifies the Epic's real prompt install hook ran by checking:
  - `state.json`
  - `plans/001-initial.md`
  - `runtime/install.json`

`claude-install-hook-fires`
- Installs the local `install-hook-epic` fixture with a real `.sh` install hook
- Verifies that the spec-defined `install` trigger ran
- Checks both:
  - `runtime/install.json`
  - `runtime/install-hook-output.json`

`claude-prompt-install-hook-fires`
- Installs the local `prompt-install-hook-epic` fixture with a real `.md` install hook
- Uses the real Claude CLI and the real Anthropic API key from `.env.e2e`
- Verifies both:
  - `runtime/install.json`
  - `runtime/prompt-hook-output.json`

`claude-can-read-installed-epic`
- Installs the remote Epic into the fixture project
- Runs Claude against the workspace
- Asserts Claude reports the installed Epic path/title/slug from real files

## Install-hook behavior

The install-hook implementation follows the local Agent Epics spec in the
sibling `agentepics` checkout, primarily `../agentepics/docs/epic-runtime.mdx`.

Current implementation choices:

- canonical trigger: `install`
- discovery precedence:
  - `hooks/install.d/`
  - `hooks/install.*`
- supported execution types in `epics install`:
  - `script`
  - `prompt`
- unsupported install-hook types fail explicitly instead of being ignored
- the CLI writes `runtime/install.json` after successful handler execution
- prompt hooks are executed through the real `claude` CLI; no fake binary or mock runtime is used

## Learnings

- Docker E2E needs two image classes:
  a lightweight CLI image and a heavier Claude image
- Required secrets for selected E2E scenarios should fail preflight, not skip.
  Silent or easy-to-miss skipping can hollow out CI coverage while the suite
  still appears green. Avoid skip-by-default for auth-dependent tests in gated
  suites unless the skip itself is the explicit behavior under test.
- Claude headless mode inside Docker needs a writable `HOME`; the harness now
  sets `HOME=/workspace/.claude-home` and `XDG_CONFIG_HOME=/workspace/.claude-home/.config`
- Remote Epic install should be tested without relying on seeded registry data,
  otherwise the test only validates local materialization
- The install-hook test needs a deterministic local fixture Epic; depending on a
  public sample repo for a hook contract is too fragile
- Separate deterministic coverage is useful for both hook styles:
  `.sh` script hooks and `.md` prompt hooks
- Claude prompts used in E2E should ask for tightly constrained output, ideally
  exact text or compact JSON, to keep assertions stable
- The harness should execute explicit programs (`epics`, `claude`) instead of
  depending on a fixed Docker entrypoint
- Run IDs need sub-second uniqueness; second-level timestamps can collide across
  concurrent local runs and break Docker image reuse

## Running

All generic scenarios:

```bash
go run ./e2e/cmd/epics-e2e run --keep-artifacts
```

Claude scenarios only:

```bash
go run ./e2e/cmd/epics-e2e run --tag claude --keep-artifacts
```

Single scenario:

```bash
go run ./e2e/cmd/epics-e2e run --scenario claude-install-remote-epic --keep-artifacts
```
