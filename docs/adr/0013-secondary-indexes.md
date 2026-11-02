# ADR-0013: Secondary attribute indexes are a best-effort cache reconciled by fsck

## Status
superseded by ADR-0014 (indexes move to the git-ignored local cache, which
removes the committed-index merge-desync hazard this ADR had to manage)

## Context
The stakeholder wants additional entrypoints into the ref tree — inverted
indexes by attribute — so "all tracks in state X" or "all tracks of type Y" is a
directory listing instead of a full scan:

- `.gitrack/state/<state>/<uuid>`
- `.gitrack/type/<type>/<uuid>`

and changes to `state`, `kind`/type, `depends-on`, `child-of` should update
these automatically. This is the ADR-0012 reverse-cache pattern applied to
attributes. But an attribute index differs from the `refs-in/` cache in a way
that matters: the `refs-in/` cache is append-only (adding an edge adds a file),
whereas an attribute index entry is a **move** — changing `state` from `open` to
`done` must DELETE `state/open/<uuid>` and CREATE `state/done/<uuid>`.

Moves are where git merges misbehave. We tested it (`tmp/index-merge.sh`):

- When two branches move the SAME track to DIFFERENT states, the authoritative
  `state` file correctly **conflicts** — but the derived index silently ends up
  with the track in **both** buckets (both adds apply, both `open` deletes
  apply). The index is corrupt precisely when the source field conflicts.
- When two branches move DIFFERENT tracks, the merge is clean and the index is
  correct.

So the index cannot be trusted as authoritative across merges, even though it is
committed and shared.

## Decision
We will support secondary attribute indexes as **committed, derived, best-effort
caches**, one directory per index, one empty file per member:

- `.gitrack/state/<state>/<uuid>` — tracks currently in `<state>`.
- `.gitrack/type/<kind>/<uuid>` — tracks of `<kind>` (ADR-0006 "type"/kind).
- the existing `refs-in/<rel>/` (ADR-0012) is the relationship form of the same
  idea; `depends-on`/`child-of` changes already maintain it.

Rules that make this safe:

1. **The authoritative field always wins.** `.gitrack/<uuid>/state` is the truth;
   `.gitrack/state/*/` is a hint. A correctness-sensitive reader confirms an
   index hit against the track's own field, or only trusts the index when the
   tree is `fsck`-clean.
2. **A normal single-branch write keeps the index exact** (move the field, move
   the bucket entry, one commit).
3. **A merge may leave the index dirty.** A conflicted field merge leaves the
   index inconsistent by construction; even some clean merges can. Therefore
   `gitrack fsck` (ADR-0012) is extended to reconcile ALL secondary indexes from
   the authoritative fields: `fsck` reports a track indexed under the wrong/extra
   bucket or missing from its bucket, and `fsck --write` rebuilds every index
   from the per-track fields. Resolving a merge conflict is followed by `fsck
   --write` (the CLI runs it automatically after a merge it detects touched
   `.gitrack/`; a git `post-merge` hook is offered for hand-merges).
4. **Indexes never gate correctness of the model.** Everything an index
   accelerates is also computable by scanning per-track fields; the index is
   only speed. If it is absent or stale, results are still obtainable (slower)
   and `fsck` restores the fast path.

## Consequences
- Easier: list-by-state and list-by-type are O(members), a directory read.
- Easier: boards/roadmaps and the web/issues generators get their primary
  groupings for free.
- Harder: the index can desync on merges. We accept this and make `fsck` the
  authority; we never let a tool trust the index over the field. This is the
  explicit cost of a committed (shared) index versus a purely-derived one.
- Harder: more files change per state transition (field + bucket move). Bounded
  and local; merges of independent transitions stay clean (tested).
- Cost accepted: redundancy (each track's state stored in its field AND its
  bucket). Fully rebuildable.

## Verification
- [ ] `.gitrack/state/<state>/<uuid>` and `.gitrack/type/<kind>/<uuid>` exist as
      one-file-per-member directories.
- [ ] A single-branch state/kind change moves both the field and the bucket
      entry so the index is exact.
- [ ] `tmp/index-merge.sh` (or its committed equivalent) demonstrates: divergent
      same-track moves conflict on the field and leave the index needing repair;
      independent moves stay clean and correct.
- [ ] `gitrack fsck` reports index drift and `fsck --write` rebuilds every
      secondary index from the authoritative per-track fields.
- [ ] No reader treats a secondary index as authoritative over a track's own
      field.
