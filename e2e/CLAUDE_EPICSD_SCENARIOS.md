# Claude epicsd Scenario Additions

These scenarios must be implemented and tested in order.

Process rules:

1. Implement scenario 1 and run it by itself.
2. Use the artifacts, failures, and any required code changes from scenario 1 before starting scenario 2.
3. Implement scenario 2 and run it by itself.
4. Use the artifacts, failures, and any required code changes from scenario 2 before starting scenario 3.
5. Implement scenario 3 and run it by itself.
6. After all three pass individually, run the new scenarios together.

## Scenario 1: `claude-epicsd-cron-state-progression`

Tags: `claude`, `live`, `daemon`, `cron`, `stateful`

What it tests:

- Claude actually advances Epic state across multiple cron heartbeats.

How it works:

1. Install an Epic with a 3-step plan:
   - step 1 creates `output/step1.txt`
   - step 2 reads `output/step1.txt` and creates `output/step2.txt`
   - step 3 reads both prior files and creates `output/summary.txt`
2. Configure cron at 15-second intervals.
3. Each daemon dispatch uses `epics resume` context, including `nextStep`.
4. Claude must read state, do the current step's work, then update runtime state so the next heartbeat advances.
5. Run for about 90 seconds.
6. Assert all 3 output files exist and later files reference earlier ones.
7. Assert the runtime state shows plan progression and is not stuck on step 1.

Why it matters:

- The existing haiku cron test only proves repeated dispatch of a stateless prompt.
- This scenario proves the stateful loop works across heartbeats.

Key cheat removed:

- A trivial “write one poem and stop” prompt.

## Scenario 2: `claude-epicsd-cron-overlap-skip`

Tags: `claude`, `live`, `daemon`, `cron`, `overlap`

What it tests:

- The daemon's overlap strategy prevents concurrent execution when a run is still in flight.

How it works:

1. Install an Epic whose prompt makes Claude wait for about 20 seconds before finishing.
2. Configure cron at `*/5 * * * * *` with `--overlap skip`.
3. Run for about 45 seconds.
4. Assert the number of actually started runs is far fewer than 9.
5. Assert skipped cron ticks are visible in persisted run data and in the daemon log.
6. Assert no two runs for the same route overlap in their `startedAt`/`finishedAt` windows.

Why it matters:

- Overlap handling is critical for production safety.

Key cheat removed:

- An instant-completion task that never triggers concurrency controls.

## Scenario 3: `claude-epicsd-webhook-auth-rejection`

Tags: `claude`, `daemon`, `webhook`, `auth`, `negative`

What it tests:

- The webhook auth layer rejects unauthorized requests and only processes a correctly authenticated request.

How it works:

1. Install an Epic, start `epicsd`, and register the workspace.
2. Create a webhook route with `--auth bearer --secret correct-token`.
3. Send a webhook with no `Authorization` header and assert `401` or `403`.
4. Send a webhook with `Bearer wrong-token` and assert `401` or `403`.
5. Send a webhook with `Bearer correct-token` and assert `202`.
6. Wait for the authenticated run to succeed.
7. Assert only the authenticated request produced an accepted dispatchable run.

Why it matters:

- The existing webhook test only covers the happy path.

Key cheat removed:

- `--auth none`.
