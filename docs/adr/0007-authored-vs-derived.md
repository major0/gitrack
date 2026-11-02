# ADR-0007: Authored data is stored; derived data is computed, never written back

## Status
superseded by ADR-0009

## Context
Three requested features look like they want a write to trigger another write:

1. When a track's `state` closes, auto-check a task-list item in any track that
   depends on it.
2. Populate a track's `referenced-by/` whenever a git commit references it.
3. (Related) keep rendered views in sync with underlying state.

A write-triggered-write is a direct threat to the merge model proven in
ADR-0003. If closing track A rewrites track B's `description` (to tick a box) or
A's `referenced-by/` (because a commit mentioned it), then:

- an edit to B's description on another branch now *false-conflicts* with, or is
  silently clobbered by, the unrelated act of closing A; and
- the stored `referenced-by/` duplicates information git history already holds,
  and must be kept consistent on every commit — exactly the derived-data trap
  our own lessons warn against (fix tooling, recompute from source of truth;
  never mutate derived copies).

We verified `referenced-by` is fully recoverable from history: `git log
--grep=gitrack:<id>` returns precisely the referencing commits and nothing else
(`tmp/ref-scan.sh`). So storage buys nothing but a consistency burden.

## Decision
We will draw a hard line between **authored** and **derived** data:

- **Authored** (the source of truth, stored as files, conflicts on divergent
  edit per ADR-0003): `kind`, `title`, `description` prose, `state`, `labels/`,
  `assignees/`, `comments/`, `attachments/`, and the relationship link
  directories (`depends-on/`, `child-of/`, `related/`, `duplicate-of/`).
- **Derived** (never stored as truth; computed on demand at query/render time):
  `referenced-by` (from `git log --grep`), the unblocked/blocked status, epic
  rollup/progress, inverse edges (`blocks`, `parent-of`), and the *checked/
  unchecked rendering* of any task-list item that is backed by a dependency.

No gitrack operation writes derived data back into an authored file. Changing a
`state` file changes exactly one file — nothing cascades on disk. All the
"automatic updates" the stakeholder wants happen at **render/query time** by
reading current authored state, so they are always correct and never a merge
hazard. A `.gitrack/index/` cache may memoize derived results for speed but is
git-ignored and always regenerable.

## Consequences
- Easier: `state` change stays a one-file local edit; merge semantics (ADR-0003)
  are preserved exactly.
- Easier: `referenced-by` and rollups are always accurate — they read live state,
  can never drift, and need no reconciliation on merge.
- Easier: no two-sided updates to keep consistent; one source of truth each.
- Harder: a reader/renderer must *compute* these views rather than `cat` a file.
  This is slightly smarter code than a bare field read, but it is pure
  (same inputs → same output) and cacheable. We accept it as the price of not
  corrupting the merge model.
- Harder: a task-list item that should reflect a dependency must be *linked* to
  that dependency (ADR-0008), not a free-text box, so the renderer knows what to
  compute. Plain prose checkboxes remain hand-toggled.

## Verification
- [ ] Closing a track modifies only that track's `state` file; no other track's
      files change on disk as a side effect.
- [ ] `referenced-by` for a track is produced by scanning `git log`, with no
      stored `referenced-by/` directory as source of truth.
- [ ] Blocked/unblocked, rollup, and inverse edges are computed, not stored.
- [ ] Any cached derived data lives under a git-ignored `.gitrack/index/`.
