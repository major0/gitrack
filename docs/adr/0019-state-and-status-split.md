# ADR-0019: Two axes — fixed `state` (open/closed) and customizable `status`

## Status
accepted

## Context
ADR-0017/0018 modeled the lifecycle as a single `state` field with three
built-ins (`new`/`closed`/`reopened`) plus user-defined flow states. The
stakeholder now wants the two concerns the field conflated to be separate,
matching the mature-tracker split (GitHub `state` open/closed; Jira
customizable `status` riding on a fixed status *category*):

- **`state`** is the fixed, never-customizable lifecycle axis: `open` / `closed`.
- **`status`** is the customizable workflow: three hard-coded built-ins plus
  user-defined statuses, with per-status transitions ("workflows") between them.

Decided values (from the stakeholder):

- `git track new` creates a track with `state=open`, `status=new`.
- states are exactly `open` and `closed` (there is no `new` state anymore).
- the three hard-coded statuses are `new`, `reopened`, `done`.

## Decision
Every track has two single-valued fields:

- **`state`** ∈ {`open`, `closed`}. Fixed set, never configurable. Authored by
  the three lifecycle verbs: `new`/`reopen` set `open`, `close` sets `closed`.
  "Closed" means exactly `state == closed`.
- **`status`** — the customizable workflow position. Three built-in statuses are
  always valid: `new`, `reopened`, `done`. Additional statuses are user-defined
  files `config/statuses/<name>` whose lines are that status's allowed next
  statuses; the three built-ins may also declare their next statuses. An
  absent/empty file means unrestricted. This replaces the old
  `config/states/<name>` workflow directory.

Verb semantics:

- `git track new` -> `state=open`, `status=new`.
- `git track update --status=<s>` -> moves `status` within the workflow,
  enforcing the current status's allowed-next set.
- `git track close [--status=<s>]` -> `state=closed`; sets `status` to the given
  status or defaults to the built-in `done`. (Closing with a custom terminal
  status, e.g. `wontfix`, is `close --status=wontfix`.)
- `git track reopen` -> `state=open`, `status=reopened`.

`state` and `status` are independent fields on disk, but the lifecycle verbs
keep them coherent (close always sets `state=closed`; new/reopen set `open`).
There is no separate close-"reason" bolt-on (ADR-0018) — the closing *status*
now carries that meaning natively.

Secondary indexes (ADR-0014 cache) index BOTH axes:
`cache/state/<state>/<uuid>` and `cache/status/<status>/<uuid>`.

## Consequences
- Easier: the model matches what users know from GitHub/Jira — a coarse
  open/closed `state` and a rich, team-defined `status` workflow on top.
- Easier: "all open" / "all closed" is a `state` query; "everything in review"
  is a `status` query; they can't contradict because close/new/reopen set both.
- Easier: the close status is a first-class workflow value, not a side metadatum.
- Harder: two fields instead of one; readers and the cache handle both. Modest.
- Harder: supersedes the ADR-0017/0018 single-field model and the
  `config/states/` layout becomes `config/statuses/`. Pre-1.0, no migration.
- Cost accepted: a track could be hand-edited into an incoherent pair
  (`state=open`, `status=done`); `fsck` can warn, and the verbs never produce it.

## Verification
- [ ] `git track new` yields `state=open` and `status=new`.
- [ ] `state` is only ever `open` or `closed`; there is no `new` state.
- [ ] `new`, `reopened`, `done` are always-valid statuses with no config.
- [ ] `config/statuses/<name>` defines a status's allowed next statuses; absent/
      empty = unrestricted; `update --status` enforces it.
- [ ] `close` sets `state=closed` and `status` (default `done`, or `--status`);
      `reopen` sets `state=open`, `status=reopened`.
- [ ] The cache indexes both `cache/state/<state>/` and `cache/status/<status>/`.
