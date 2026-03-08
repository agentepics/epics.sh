# E2E Harness

The `e2e/` subsystem runs Docker-backed end-to-end scenarios against the real
`epics` CLI binary built from this repo.

## Requirements

- Docker CLI
- a reachable Docker daemon

## Common commands

```bash
go run ./e2e/cmd/epics-e2e list
go run ./e2e/cmd/epics-e2e run
go run ./e2e/cmd/epics-e2e run --scenario install-local-fixture
go run ./e2e/cmd/epics-e2e run --tag claude
go run ./e2e/cmd/epics-e2e run --keep-artifacts
go run ./e2e/cmd/epics-e2e run --json
```

Artifacts are written under `.e2e-artifacts/`.

Each run writes:

- `run.log` for top-level harness operations
- `run.events.jsonl` for structured top-level events
- `summary.json` for machine-readable results
- one `operations.log` plus one `operations.events.jsonl` per scenario
- one `*.operations.log` plus `*.operations.events.jsonl` per step
- workspace manifest snapshots before preparation, after preparation, before each step, after each step, and at scenario end

Assertion logs now include expected and observed values or previews so you can
see exactly what was checked.

Passing runs keep the logs even when workspace snapshots are cleaned up.

## CI

GitHub Actions runs the harness with:

```bash
go run ./e2e/cmd/epics-e2e run --keep-artifacts
```

Artifacts are uploaded from `.e2e-artifacts/` even when the job fails.
