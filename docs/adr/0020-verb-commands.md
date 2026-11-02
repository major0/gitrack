# ADR-0020: Every CLI command is a verb — `create`, not `new`

## Status
accepted

## Context
ADR-0018 made `new` the canonical create verb so the create/close/reopen
commands mirrored the then-states `new`/`closed`/`reopened`. ADR-0019 then split
the model: `new` is now a *status value*, not a lifecycle state, and the fixed
states are `open`/`closed`. That removes the reason `new` was a command name — a
command called `new` no longer corresponds to any state transition verb, and it
reads as a noun among otherwise-verb commands (`update`, `show`, `close`, ...).

## Decision
Every CLI command is a verb. The create command is **`create`**; `new` is **not**
a command (no alias). The full verb set: `init`, `create`, `update`, `show`,
`list`, `search`, `close`, `reopen`, `fsck`. `new` survives only as a *status
value* (ADR-0019), never as a command.

## Consequences
- Easier: a consistent, verb-only command surface; no noun/verb mix.
- Easier: no confusion between the `new` status and a `new` command.
- Harder: reverses ADR-0018's canonical `new`. Pre-1.0, no external users; the
  change is a one-line dispatch edit. Accepted.
- Cost accepted: anyone who learned `git track new` (there is no such audience
  yet) uses `git track create`.

## Verification
- [ ] `git track create <title>` creates a track; `git track new` is an unknown
      command.
- [ ] All commands in `--help` are verbs.
- [ ] `new` appears only as a status value, never as a command.
