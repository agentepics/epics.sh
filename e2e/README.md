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
go run ./e2e/cmd/epics-e2e chat
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

## Live Chat

`chat` is a manual live-evaluation workflow built on the same Claude Docker
image used by the live scenarios. It:

- builds the Claude E2E image
- starts a detached container with `claude` and `epics` installed
- mounts a prepared workspace under `/workspace`
- installs the `examples/fixtures/resume-epic` fixture into the project
- runs a scripted multi-turn conversation with Claude via `docker exec`
- writes a transcript plus host-doctor output under `.e2e-artifacts/`

Example:

```bash
go run ./e2e/cmd/epics-e2e chat
```

The command prints:

- the artifact directory
- the running container name
- a transcript path
- a `docker exec -it ... bash` shell command
- a cleanup command

To continue exploring manually after the scripted chat:

```bash
docker exec -it <container-name> bash
docker exec -it <container-name> claude
```

To remove the container automatically when the scripted chat ends:

```bash
go run ./e2e/cmd/epics-e2e chat --cleanup
```

## CI

GitHub Actions runs the harness with:

```bash
go run ./e2e/cmd/epics-e2e run --keep-artifacts
```

Artifacts are uploaded from `.e2e-artifacts/` even when the job fails.
