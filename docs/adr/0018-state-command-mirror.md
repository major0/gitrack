# ADR-0018: Subcommands mirror the built-in states; close qualifier is `status`

## Status
superseded by ADR-0019 (the close qualifier becomes the first-class `status`
field; `new` is no longer a state). The `new` canonical verb and the
command/state mirror intent are retained there.

## Context
ADR-0017 fixed the state model: three built-in states (`new`, `closed`,
`reopened`), user-defined flow states, and a close *qualifier* stored as
metadata rather than as a state. It named that qualifier field `reason` and the
close flag `--reason`.

The stakeholder refined two things:

1. The managing subcommands should mirror the built-in states, so the command
   you run names the state you land in: `git track new` -> `new`, `git track
   close` -> `closed`, `git track reopen` -> `reopened`. (Previously the create
   verb was canonically `create` with `new` as an alias.)
2. The close qualifier is spelled `--status`, i.e. `git track close
   --status=<status> <ref>`.

This is a naming/ergonomics refinement, not a change to the state model. The
qualifier is still metadata on the single `closed` state, not a state.

## Decision
- The canonical create verb is **`new`** (with `create` kept as an alias), so
  the three state-changing verbs `new` / `close` / `reopen` mirror the three
  built-in states `new` / `closed` / `reopened` one-to-one.
- The close qualifier field is renamed from `reason` to **`status`**, set via
  `git track close --status=<status> <ref>` (also accepting `--status <value>`
  and `-s <value>`). It records why/how a track closed (e.g. `wontfix`,
  `duplicate`, `completed`) and remains separate metadata — never a state.
  `reopen` clears the `status` field.

Everything else in ADR-0017 (one `state` field, built-in trio,
`config/states/<name>` transitions, enforcement) is retained unchanged.

## Consequences
- Easier: the CLI is mnemonic — the subcommand is the resulting state. Less to
  remember; matches the stakeholder's mental model.
- Easier: `--status` aligns with the vocabulary the stakeholder uses for the
  close qualifier.
- Harder: the stored field is `status`, which is a word ADR-0017 had avoided to
  keep "no separate status *field* that is itself a lifecycle state." We accept
  the name because here `status` is explicitly a *qualifier on closed*, not a
  parallel lifecycle; the single lifecycle field remains `state`. Documented so
  the distinction is not lost.
- Cost accepted: a track closed before this rename would carry `reason`; pre-1.0
  with no real data, so no migration is written.

## Verification
- [ ] `git track new` is canonical; `create` still works as an alias.
- [ ] `new`/`close`/`reopen` land a track in `new`/`closed`/`reopened`.
- [ ] `git track close --status=wontfix <ref>` sets `state=closed` and writes a
      `status` file containing `wontfix`; `--status <v>` and `-s <v>` also work.
- [ ] `reopen` sets `state=reopened` and removes any `status` file.
- [ ] No `reason` field is written by `close`.
