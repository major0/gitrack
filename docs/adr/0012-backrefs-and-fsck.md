# ADR-0012: Backward references as a committed, mergeable cache; `gitrack fsck`

## Status
superseded by ADR-0014 (the backref index moves to the git-ignored cache; the
`fsck`-rebuilds-from-forward-links principle is retained)

## Context
ADR-0009 made `referenced-by` and inverse edges purely *derived* (computed from
forward links and git history, never stored) to avoid a write-on-write merge
hazard. The stakeholder now wants forward/backward references stored in each
repo "to make it fast to find/update," plus a `gitrack fsck` to locate and
repair the ref tree.

Stored backrefs appear to conflict with ADR-0009. We tested the actual hazard
(`tmp/backref-merge.sh`): the danger is *shape*, not storage. Backrefs written
as a single aggregate file both sides append to → merge **conflict**. Backrefs
written as a **directory with one file per referrer** → clean **union merge**,
identical to `depends-on/` and every other multi-value dir we already ship. So
stored backrefs are safe to commit *iff* they use the one-file-per-entry shape.

`git fsck` "verifies the connectivity and validity of the objects in the
database" — a validator against authoritative content, not a source of truth.
That is exactly the role we want for `gitrack fsck`: rebuild and repair the
backref cache from the authoritative forward links.

## Decision
Forward links remain the single source of truth (`depends-on/`, `child-of/`,
`related/`, `duplicate-of/` on the referencing track). We add a **materialized,
committed, mergeable backref cache** so lookups and updates are fast without a
full-repo scan:

- Each track gains `refs-in/<rel>/<referrer-uuid>` — a directory per relation
  type, one empty file per track that points at this one. It is the reverse of
  the forward links, stored in the one-file-per-entry shape proven to merge as a
  clean union.
- The cache is **committed** (shared on clone), not git-ignored. It is still
  *derived*: it is a function of the forward links and must never be treated as
  the source of truth. ADR-0009's rule — no write into *authored* data as a side
  effect — is preserved: adding a forward link writes the referrer's authored
  `depends-on/<t>` AND the target's `refs-in/...` cache entry; these are two
  distinct files, and concurrent additions union-merge (tested). A `state`
  change still writes only `state`.
- **`gitrack fsck`** is the validator/repair tool. It walks all forward links,
  recomputes the expected `refs-in/` tree, and reports + optionally fixes drift
  (missing backrefs, stale backrefs whose forward link is gone, dangling refs to
  nonexistent tracks, and unresolvable/untracked cross-repo refs per ADR-0011).
  `fsck --write` rebuilds the cache from scratch; a fresh clone or a hand-edited
  tree is made consistent by one `fsck`.

This refines ADR-0009 (it does not supersede it): the authored/derived split
stands; `referenced-by` from git history (structural + trailer) stands; we add a
materialized cache for the *relationship* backrefs whose correctness `fsck`
guarantees.

## Consequences
- Easier: "what blocks on this / what is this a child of" is a directory listing,
  O(edges-into-this-track), not a full-repo scan.
- Easier: the cache is shared on clone, so tools are fast immediately without a
  rebuild step; `fsck` is only needed after hand-edits or to verify.
- Easier: the one-file-per-entry shape means concurrent edits union-merge; no new
  merge hazard (empirically verified).
- Harder: a forward-link write now also writes a backref file, so the writer must
  touch two tracks' dirs. If a write lands the forward link but not the backref
  (crash, hand-edit, bad tool), the cache drifts — `fsck` detects and repairs it.
  The cache is never trusted over forward links when they disagree.
- Harder: cross-repo backrefs (ADR-0011) can only be cached for reachable
  remotes; `fsck` marks unreachable ones rather than failing.
- Cost accepted: a little redundancy on disk (each edge stored twice, forward +
  reverse). Bounded by edge count and fully rebuildable.

## Verification
- [ ] `refs-in/<rel>/<referrer-uuid>` exists as a one-file-per-entry directory;
      concurrent additions on two branches union-merge with no conflict
      (`tests/merge-semantics.sh` extended).
- [ ] Adding a forward link also creates the matching `refs-in/` entry on the
      target; removing it removes the entry.
- [ ] `gitrack fsck` reports drift between forward links and `refs-in/`, and
      `fsck --write` makes them consistent; forward links win on disagreement.
- [ ] `fsck` flags dangling refs (to nonexistent tracks) and unresolvable
      cross-repo refs.
- [ ] A fresh clone with no `refs-in/` is made correct by one `fsck --write`.
