# ADR-0005: Relationship edges and their scheduling direction

## Status
accepted

## Context
ADR-0004 established one stored relationship primitive (typed link dirs,
inverses computed) and claimed `child-of` is "a `depends-on` edge that also
groups" — both treated as the source being blocked by the target.

Building the worked example (`docs/EXAMPLE.md`) proved that claim directionally
wrong. The two edges block in *opposite* directions:

- `A depends-on B` — A waits for B. A is blocked until B is closed.
- `A child-of B` — B (the epic/parent) waits for A (the child). A parent is
  blocked until all its children are closed; the child is **not** blocked by the
  parent. This is the inverse of `depends-on`.

Lumping them as the same blocking edge would make every child spuriously
"blocked by" its epic, which is backwards.

## Decision
We keep ADR-0004's storage model unchanged (typed link directories
`.gitrack/<id>/<rel>/<target-id>`; inverse edges computed, not stored). We
correct the **scheduling semantics**:

Each stored edge contributes to a single directed **blocking graph** whose edge
means "blocker → blocked" (an arrow points at the thing that must wait):

| stored edge (on A, pointing at B) | blocking-graph edge | reading |
|-----------------------------------|---------------------|---------|
| `A depends-on B`                  | B → A               | A waits for B |
| `A child-of B`                    | A → B               | B (parent) waits for A (child) |
| `A blocks B` (not stored; inverse of depends-on) | A → B | B waits for A |
| `A related C`                     | none                | no scheduling effect |
| `A duplicate-of D`                | none                | no scheduling effect |

So `depends-on` and `child-of` are still ONE primitive (a directed link) and the
graph builder is still dumb, but it adds the edge in the correct direction based
on the link type: `depends-on` reversed, `child-of` as-is. Containment/rollup
still reads `child-of` only; it is unaffected by direction.

A track is **unblocked** iff it has no incoming blocking-graph edge from an
*open* track — i.e. for `A`: every `B` where `A depends-on B` is closed, AND
every child `C` where `C child-of A` is closed. Leaf tracks with no `depends-on`
and no open children are unblocked.

## Consequences
- Easier: the unblocked-work frontier and epic rollup are both correct and come
  from one traversal that branches only on link type → direction.
- Easier: an epic auto-closes-eligible exactly when its children are all closed;
  a child is actionable regardless of its epic's state.
- Harder: nothing material; the storage is identical to ADR-0004. Only the graph
  builder's edge-direction rule changed.
- Cost accepted: readers must know the direction table above; it lives in
  FORMAT.md §4 so there is one source of truth.

## Verification
- [ ] Graph builder adds `depends-on` as a reversed edge and `child-of` as a
      forward edge (blocker → blocked).
- [ ] In the worked example, `issue-session` is blocked (its `depends-on`
      target `issue-login` is... closed) → actually unblocked; `epic-auth` is
      blocked while any child is open.
- [ ] A child track is never reported blocked solely because its parent epic is
      open.
- [ ] `related` and `duplicate-of` contribute no blocking edge.
