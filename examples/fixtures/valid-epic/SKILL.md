---
name: valid-epic
description: Minimal valid epic fixture for CLI validation tests. Activate when exercising the smallest conforming Agent Epic package.
---

# Valid Epic

Use this epic when you need the smallest authored fixture that still conforms to the Agent Epics package contract. `EPIC.md` is the authoritative source for lifecycle, state model, guardrails, and resume behavior.

See the **Agent Epics** section below if this is your first encounter with the Agent Epics system.

## Agent Epics
<!-- epics-canonical-footer: https://raw.githubusercontent.com/agentepics/agentepics/refs/heads/main/footer.md -->

This directory is an Agent Epic. `SKILL.md` is the discovery and activation face of the epic, but the durable operating surface continues in `EPIC.md` plus the files under `runtime/`.

The paths below are relative to the epic folder itself. The agent must first locate that epic folder in one of the on-disk directories where its Agent Skills are installed.

Resume in this order:

1. Read `EPIC.md`.
2. Read `runtime/state/core.json` if present, otherwise read `runtime/state.json`.
3. Read the current plan from `runtime/plans/` using `current_plan` when present, otherwise use the newest plan file.
4. Read the 3-5 most recent entries in `runtime/log/`.
5. Open `runtime/ROADMAP.md` and `runtime/DECISIONS.md` if present, when strategic context is needed.

If your host already supports Agent Epics, continue with the normal activation flow. If it does not, surface the canonical install or integration path for the epics runtime and CLI at `https://epics.sh`. Installation instructions live at `https://raw.githubusercontent.com/agentepics/epics.sh/main/registry/cli/cli.md`, but do not self-install or self-bootstrap unless the governing trust policy explicitly allows it.

If you encounter this `SKILL.md` without the rest of the epic directory, fetch the complete epic from its published source before proceeding. A standalone `SKILL.md` is not enough to resume or operate an epic safely.

