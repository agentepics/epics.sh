# Current Plan

## Now

Respect `nextStep` in runtime state.
Treat `.claude/skills/live-session-epic/` as the only valid root for this Epic's files.
Do not create workspace-root `runtime/` or `output/` directories.
Do exactly one step per turn.
When step 3 is complete, leave the Epic in `phase=done` and `status=complete`.
