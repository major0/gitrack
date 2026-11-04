# Concepts

A short tour of the ideas that make gitrack different from a hosted tracker.
Understanding these four makes every command obvious.

## 1. The tracker lives in your working tree

gitrack stores the tracker as plain files under `.gitrack/` in your repository's
working tree — not in a database, not in a separate git ref, not on a server.
The practical consequences:

- **It merges with your code.** A branch that changes an issue and the code that
  fixes it merges as one ordinary `git merge`. There is no second system to
  reconcile.
- **It travels everywhere git goes.** A clone, a `git bundle` on a USB stick, an
  S3-backed remote, a shared drive — if it moves git, it moves the whole
  tracker, with full history.
- **It is readable without the tool.** Every field is a file. You can `cat`,
  `grep`, and edit the tracker with ordinary tools; gitrack is a convenience
  over the format, not a gatekeeper.

## 2. One object: the track

There is a single object type — the **track**. An epic, a project, an issue, a
bug, a task, and a note are all tracks that differ only by a cosmetic `kind`
field. This keeps the model tiny: everything you learn about creating, reading,
linking, and closing a track applies to all of them.

Each track has a stable **UUID** (a UUIDv7) and a human-chosen **slug**. You
reference a track by UUID, short UUID, or slug — see
[References](references.md).

## 3. Two lifecycle axes: state and status

A track's lifecycle is two separate fields, and keeping them separate is a
deliberate design choice:

- **`state`** is fixed and never customizable: `open` or `closed`. "Closed"
  means exactly `state == closed`. Only `create`, `close`, and `reopen` change
  it.
- **`status`** is a customizable workflow position. Three statuses are built in
  and always valid — `new`, `reopened`, `done` — and you can define your own
  (`triaged`, `in-progress`, `in-review`, ...) with allowed transitions between
  them. `update -s <status>` moves it.

So `state` answers "is this done?" for boards and rollups, while `status`
answers "where is this in our process?" You shape `status` to your team;
`state` stays universal. See [Status workflows](workflow.md).

## 4. Smart data, dumb code

The structure lives in the data, so the code stays simple and the derived views
are always recomputed, never stored:

- A single-valued field is a **file** (`title`, `state`, `status`). Two people
  setting it differently produce a git conflict — a human resolves it. That is
  intended.
- A multi-valued attribute is a **directory** (`labels/`, `comments/`,
  `depends-on/`). Concurrent additions are different files and merge as a clean
  union.
- Anything that can be computed is computed: whether a track is blocked, who
  references it, a board's Ready/Blocked split, a checklist's checked state.
  gitrack keeps a local, git-ignored cache for speed, but the authored files are
  always the source of truth, and `git track fsck` rebuilds the cache from them.

Because derived data is never committed, it never travels through a merge and
can never desync. The committed tree is purely authoritative.

## How the pieces fit

You author tracks, relationships, comments, and config. gitrack commits each
change as its own `.gitrack/`-scoped revision under your git identity, maintains
the local cache, and renders derived views (boards, the website, autolinked
references). The [command reference](../reference/commands.md) lists every verb;
the next pages cover each subject in depth.
