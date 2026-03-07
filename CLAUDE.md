# CLAUDE.md

`epics.sh` is the umbrella repo for the website and the `epics` CLI.

The Epic spec itself lives in:

- https://github.com/agentepics/agentepics

Keep these rules in mind:

- the Epic format stays portable and spec-first
- the `epics` CLI is the core implementation surface
- host integrations are adapters around the CLI
- runtime-heavy features should stay optional unless deliberately promoted

Useful docs:

- `docs/planning/ROADMAP.md`
- `docs/architecture/DAEMON.md`
- `docs/specification/SPEC_EXTENSION.md`
- `docs/adapters/README.md`
