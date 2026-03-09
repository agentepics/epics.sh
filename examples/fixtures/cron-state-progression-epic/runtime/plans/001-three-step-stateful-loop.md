# Current Plan

This Epic must advance through three separate cron heartbeats.

## Now

Respect `nextStep` in runtime state.
Open `runtime/state/core.json` and use the explicit `step_instructions` plus `step_transitions`.
Treat `.claude/skills/cron-state-progression-epic/` as the only valid root for this Epic's files.
Do not create workspace-root `runtime/` or `output/` directories.
Do exactly one step per run.
When step 3 is complete, leave the Epic in `phase=done` and `status=complete`.
