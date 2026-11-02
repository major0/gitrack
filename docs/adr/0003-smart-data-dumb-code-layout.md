# ADR-0003: Filesystem-as-database — one file per metadata field

## Status
accepted

## Context
ADR-0001 chose an append-only event log, computing state by folding events, to
avoid merge collisions. That optimizes for conflict-*avoidance*. But the
stakeholder wants the opposite for mutable state: when two developers change the
*same* track's state on different branches, that disagreement SHOULD surface as
a merge conflict a human resolves — the conflict is a feature, not a failure.

The stakeholder also wants "smart data, dumb code" pushed to the extreme: the
on-disk layout should carry the structure so the code that reads it is trivial.
Folding an event log is "smart code"; `cat .gitrack/<id>/state` is dumb code.

We pressure-tested a file-per-field layout against real `git merge` (six
scenarios, `tmp/merge-scenarios.sh`). Results, all as desired:

| concurrent edit                              | git result   |
|----------------------------------------------|--------------|
| same track `state` -> different values       | **conflict** |
| same track `state` -> same value             | clean        |
| `state` on one side, `title` on the other    | clean        |
| two different `comments/<uuid>.md` files      | clean union  |
| two different `depends-on/<id>` entries       | clean union  |
| same `description` edited two ways            | **conflict** |

The empirical behavior is exactly the desired semantics, with no code at all.

## Decision
We will store each track as a directory `.gitrack/<uuid>/`, and represent its
metadata as **one file per field**: the filename is the metadata key, the file
content is the value (`title`, `description`, `state`, `kind`, ...).
**Multi-valued metadata is a subdirectory**, one entry per file (`comments/`,
`attachments/`, and relationship directories). Reading a field is reading a
file; there is no fold, no replay, no computed state. Concurrent edits to the
same single-value field conflict by construction; appends to multi-value
directories merge as a clean union. This supersedes the event-log model of
ADR-0001 while keeping its working-tree requirement (ADR-0001 Context still
holds) and the universal-object/UUID identity of ADR-0002.

## Consequences
- Easier: the reader is `cat`. Any tool, in any language, trivially inspects a
  track. Maximum "dumb code."
- Easier: genuine disagreements on a field surface as conflicts automatically.
- Easier: independent field edits and multi-value appends merge with no fuss.
- Harder: we lose automatic causal ordering of a field's history. A single-value
  field keeps only its current value, not its change log. We accept this; where
  history matters (the discussion thread) it is modeled as append-only files in
  a directory (`comments/`), which never lose history.
- Harder: a conflict in `state` requires a human. That is the intended cost.
- Cost accepted: a track is many small files. Fine for git and for humans.

## Verification
- [ ] A track is a directory `.gitrack/<uuid>/` of single-value field files plus
      multi-value subdirectories.
- [ ] `tmp/merge-scenarios.sh` (or its committed equivalent) shows same-field
      divergent edits conflict and independent edits / multi-value appends merge.
- [ ] Reading any field is a direct file read with no folding or replay.
- [ ] No tool writes a precomputed/cached state file as a source of truth.
