# Status workflows

gitrack separates a track's lifecycle into two axes. This page explains both and
shows how to shape the customizable one with `git track config`.

## The two axes

### `state` — fixed

`state` is `open` or `closed`, and nothing else. A track is "done" exactly when
`state == closed`. This is the axis boards, rollups, and the issues view read.
It is never customizable, and only three commands change it:

- `create` → `state=open`
- `close` → `state=closed`
- `reopen` → `state=open`

### `status` — customizable

`status` is where a track sits in *your* process. Three statuses are built in
and always valid:

- `new` — set by `create`
- `reopened` — set by `reopen`
- `done` — the default set by `close`

Everything else — `triaged`, `in-progress`, `in-review`, `blocked`,
`needs-info`, `duplicate` — is yours to define. You move `status` with:

```sh
git track update <ref> -s in-progress
```

The two axes stay coherent through the lifecycle verbs: `create` gives
`open`/`new`, `close` gives `closed`/`done` (or `closed`/`<your status>` with
`--status`), and `reopen` gives `open`/`reopened`.

## Defining a workflow

A user-defined status is a file `.gitrack/config/statuses/<name>` whose lines
are that status's **allowed next statuses**. A status with no file, or an empty
one, permits any transition — enforcement is opt-in, per status.

Edit workflows with `git track config status` rather than by hand:

```sh
git track config status add triaged --next in-progress,needs-info
git track config status add in-progress --next in-review,done
git track config status add in-review --next done,in-progress
git track config status set-next triaged in-progress needs-info closed
git track config status list
git track config status rm needs-info
```

- `add <name> [--next a,b,c]` creates a status with its allowed transitions.
- `set-next <name> [next...]` replaces the allowed transitions.
- `list` shows all statuses and their transitions.
- `rm <name>` removes a user-defined status. The three built-ins
  (`new`/`reopened`/`done`) cannot be removed.

Transition targets are validated to be existing statuses, so a workflow can't
point at a status that doesn't exist.

## How enforcement works (guardrails, not a gate)

Workflows are **guardrails**, not hard enforcement. gitrack's own `update -s`
path validates the transition and rejects a disallowed move. But the tracker is
plain files that anyone can edit directly, and merges happen outside the tool,
so the config describes *intended* structure — it cannot constrain what the tree
physically contains.

`git track fsck` is the reconciler: it reports where authored data diverges from
the guardrails as advisory warnings, and never rewrites an authored field to
match config. This keeps the well-behaved CLI path consistent while respecting
that the data, not the config, is authoritative.

## A worked example

```sh
# Define a small flow: new -> triaged -> in-progress -> in-review -> done
git track config status add triaged     --next in-progress
git track config status add in-progress --next in-review,done
git track config status add in-review   --next done,in-progress

id=$(git track create "Add password reset")
git track update "$id" -s triaged
git track update "$id" -s in-progress
git track update "$id" -s in-review
git track close "$id"                 # closed/done
```

Trying `git track update "$id" -s done` while in `triaged` would be rejected,
because `triaged` only allows `in-progress`.

## Kinds (a related, cosmetic axis)

`kind` (`task`, `bug`, `epic`, ...) is a separate, purely cosmetic label. You
may declare an allowed set in `.gitrack/config/kinds` (one per line) as a
guardrail, but like statuses it is advisory. See
[Configuration](configuration.md).
