# ADR-0017: Three built-in states, user-defined flow, per-state transitions

## Status
superseded by ADR-0019 (the single `state` field is split into a fixed `state`
open/closed and a customizable `status` workflow)

## Context
FORMAT.md §11 left open whether a track needs one `state` field or a `state` plus
a `status`, and the initial config shipped an ad-hoc `states` list with a
separate `states.closed` subset (`done`/`wontfix`/`duplicate`). The stakeholder
has now specified a concrete model:

- Three states are **built-in and mandatory**: `new` (every track starts here),
  `closed` (every closed track), `reopened` (every reopened track). They always
  exist and cannot be removed.
- Every other state is a **user-defined flow step**.
- Each state (including the built-in three) declares its allowed **next
  state(s)** — defining a state is defining its outgoing transitions. This builds
  a workflow graph.

An open sub-question: is a close *reason* (wontfix, duplicate, completed) a
distinct state, or metadata on the single `closed` state? The stakeholder's
wording — exactly three built-ins, `closed` singular, other states "build a
flow" toward it — implies `closed` is a single terminal state and the reason is
separate. This matches the `gh`/GitHub model (one closed state + a state_reason).
We adopt that reading.

## Decision
A track has exactly one `state` field. There is no separate `status` field; the
earlier `states.closed` subset file is removed.

Three states are built in and always valid: **`new`**, **`closed`**,
**`reopened`**. A track is "closed" iff its state is `closed` (the single
terminal). `new` and `reopened` are open states. Creating a track forces
`state = new`; `git track close` forces `state = closed`; `git track reopen`
forces `state = reopened`.

All other states are user-defined and live in `config/states/<name>` — one file
per state (smart-data: a state is a directory entry, its transitions are its
content). Each state file lists its allowed next states, one per line; the three
built-ins may also have a `config/states/<built-in>` file to declare *their*
next states (e.g. `new -> triaged`, `closed -> reopened`), even though the states
themselves are mandatory. A state with no file (or an empty file) permits any
transition — workflow enforcement is opt-in per state.

A close **reason** (completed, wontfix, duplicate, ...) is separate metadata: a
`reason` field set at close time, not a state. This keeps the state graph small
and the "why" orthogonal to the "where in the flow."

Transition enforcement: `git track update --state X` checks that `X` is in the
current state's allowed-next set when that set is non-empty; `close`/`reopen`
are always permitted (they target built-ins) but still record the move.

## Consequences
- Easier: a universally understood lifecycle (new -> ... -> closed, reopen ->
  ...) with a tiny mandatory core, so a brand-new tracker is immediately usable.
- Easier: the workflow graph is smart-data — states and their transitions are
  files/lines; the engine that validates a move is a set-membership check.
- Easier: close reasons don't multiply the state space; filtering "all wontfix"
  is a `reason` query, "all closed" is a `state` query.
- Harder: existing config (`states` list + `states.closed`) is replaced; `init`
  and readers change. Low cost — pre-1.0, no real data yet.
- Harder: two ways to express "done vs wontfix" existed; we picked reason-as-
  metadata. Reversing to reason-as-state later would be a migration. Accepted as
  the deliberate, gh-aligned choice.

## Verification
- [ ] `new`, `closed`, `reopened` are always valid states with no config.
- [ ] `create` sets `state=new`; `close` sets `state=closed`; `reopen` sets
      `state=reopened`.
- [ ] Per-state next-states live in `config/states/<name>`; an empty/absent file
      means unrestricted; a non-empty file restricts `update --state`.
- [ ] The three built-ins may declare next-states via their own config files.
- [ ] A track is closed iff `state == closed`; close reason is a separate
      `reason` field, not a state.
- [ ] `update --state` to a disallowed next state is rejected with the allowed set.
