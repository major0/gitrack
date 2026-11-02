# ADR-0004: One relationship primitive; `child-of` is a `depends-on` that also groups

## Status
superseded by ADR-0005

## Context
The stakeholder questioned whether `depends-on` and `child-of` are two names for
one thing: "an Epic/Project that has multiple children depends-on the completion
of those children." That intuition is largely correct — for *scheduling*, a
parent is blocked by its open children exactly as if it depended on them.

Two things survive that a pure-dependency model would lose:

1. **Containment/exclusivity.** A `child-of` edge says the child *belongs to* one
   parent (an issue has one epic). A `depends-on` edge is many-to-many and
   carries no ownership — an issue can depend on any number of unrelated issues.
2. **Rollup.** "This epic is 6 of 10 done" is a containment query over children.
   Dependency gives blocking, not membership or progress aggregation.

We do not want two relationship mechanisms with duplicated graph code.

## Decision
We will store relationships as **typed links in per-type subdirectories** under
a track: `.gitrack/<id>/<rel>/<target-id>`, where `<rel>` is the relationship
type (`depends-on`, `child-of`, `related`, `duplicate-of`). The link file is
empty or carries optional note text; its name is the target UUID.

There is **one graph primitive**: a directed edge. `child-of` is defined as "a
`depends-on` edge that additionally asserts containment." The scheduling graph
treats `depends-on` and `child-of` identically (both block the source until the
target is closed). Containment and rollup read `child-of` edges *only*. One
mechanism, two semantics, no duplicated traversal code — the relationship type
is data, and the dumb graph code branches on it only where containment genuinely
differs from scheduling.

Inverse edges (`blocks`, `parent-of`) are **not stored**; they are computed by
reversing the stored edges when building the graph, so there is a single source
of truth per relationship and no two-sided update to keep consistent.

## Consequences
- Easier: adding a relationship is creating an empty file in a directory; it
  merges as a clean union with concurrent additions (proven in ADR-0003).
- Easier: one traversal builds both the unblocked-work DAG (all blocking edges)
  and the epic tree (child-of edges) from the same data.
- Harder: `child-of` exclusivity (one parent) is a *convention* the CLI enforces,
  not something the filesystem guarantees; a hand-edit could give two parents.
  We accept this and have the CLI/linter flag it rather than complicate storage.
- Cost accepted: computing inverse edges is an O(edges) pass. Negligible.

## Verification
- [ ] Relationships live at `.gitrack/<id>/<rel>/<target-id>` as files named by
      target UUID.
- [ ] The scheduling/unblocked-work graph treats `depends-on` and `child-of` as
      the same blocking edge.
- [ ] Containment/rollup queries read only `child-of` edges.
- [ ] `blocks` / `parent-of` are never stored; they are derived by edge reversal.
- [ ] A CLI check flags a track with more than one `child-of` parent.
