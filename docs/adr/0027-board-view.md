# ADR-0027: `git track board` — ready/blocked view and the dependency digraph

## Status
accepted

## Context
The original goal list calls for project boards and roadmaps with a digraph of
unblocked work computed from depends-on/child-of and track state. The graph
primitives exist (ADR-0005: Blockers/IsUnblocked), so this is a query+render
layer, not new storage.

"Board / roadmap / digraph" are three related outputs. The through-line the
stakeholder named is the **unblocked-work digraph**: the actionable frontier of
open tracks with no open blocker, plus the blocking graph that explains it. We
want one view surface, not three commands.

A naive implementation calls Blockers() per track; since Blockers() scans all
tracks for child-of edges, that is O(N^2). The view must build the graph once.

## Decision
Add `git track board` with a text default and a graph mode:

- `git track board` (text): list OPEN tracks in two groups — **Ready** (unblocked)
  and **Blocked** (each with the open blockers that hold it). Closed tracks are
  omitted (they are not work). This is the actionable board/roadmap.
- `git track board --graph` (alias `--mermaid`) / `--dot`: emit the blocking
  **digraph** in a standard format. Mermaid is the default because it renders
  directly on GitHub and in the future static site; DOT (`--dot`) serves
  Graphviz. Edges point blocker -> blocked (an arrow points at what waits);
  ready (unblocked, open) nodes are marked so the frontier is visible.

Core gains a single `Board()` pass that is O(N + edges): load every track once,
build the child-of reverse index once, then classify. All board/graph output
derives from that one computation; no per-track rescans.

Scope held for a first slice: no per-status kanban columns and no milestone
grouping (status buckets already answer "group by status" via the cache). Those
are additive later; this ADR delivers the ready/blocked frontier and the
digraph, which is the headline deliverable.

## Consequences
- Easier: one command answers "what can I work on now?" (Ready) and "what is
  waiting, and on what?" (Blocked), and emits a shareable digraph.
- Easier: Mermaid output drops into Markdown/GitHub and the planned web view.
- Easier: O(N + edges) keeps the board fast on large trackers.
- Harder: a cyclic depends-on graph has no fully-ordered frontier; we define
  Ready purely as "no OPEN blocker", which is well-defined even with cycles, and
  the digraph shows the cycle rather than erroring.
- Cost accepted: two graph formats to maintain (Mermaid default, DOT flag).

## Verification
- [ ] `git track board` lists open tracks under Ready/Blocked; closed tracks are
      omitted; each Blocked track shows its open blockers.
- [ ] A child is Ready even when its parent epic is open; an epic is Blocked
      while any child is open (ADR-0005 directions).
- [ ] `--graph`/`--mermaid` emits valid Mermaid; `--dot` emits valid DOT; both
      orient edges blocker -> blocked and mark ready nodes.
- [ ] Board computation is a single pass (no per-track Blockers rescan).
