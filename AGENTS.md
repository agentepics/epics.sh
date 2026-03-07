# AGENTS.md

`epics.sh` is one repo for:

- the `epics.sh` website
- the `epics` CLI

The Epic spec itself lives in:

- https://github.com/agentepics/agentepics

Public curated sample Epics authored by the project should live in:

- https://github.com/agentepics/epics

Rules:

- keep the Epic model host-neutral
- treat `epics` CLI as the canonical behavior surface
- keep daemon/runtime concepts optional unless explicitly documented otherwise
- use relative links and do not commit local absolute paths

Key paths:

- `apps/web/`
- `cmd/epics/`
- `internal/`
- `examples/`
- `registry/`
- `docs/`

Read before major architectural changes:

- `docs/planning/ROADMAP.md`
- `docs/planning/WEB_PRD.md`
- `docs/architecture/DAEMON.md`
- `docs/specification/SPEC_EXTENSION.md`
- `docs/adapters/README.md`
