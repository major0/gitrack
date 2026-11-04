# Boards and roadmaps

The board turns your tracks and their relationships into a view of what is
**actionable right now** versus what is **waiting**, plus a dependency graph you
can render anywhere.

## The Ready/Blocked board

```sh
git track board
```

Open tracks are split into two groups:

- **Ready** — no open blocker. These are actionable now.
- **Blocked** — waiting on something, shown with the open blockers that hold
  them.

Closed tracks are omitted — the board is about remaining, unblocked work.
"Blocked" is computed from relationships exactly as described in
[Relationships](relationships.md): a track is ready when every `depends-on`
target is closed and every child is closed.

## The dependency digraph

The same command emits the blocking graph for rendering elsewhere:

```sh
git track board --mermaid     # Mermaid source (default graph form)
git track board --dot         # Graphviz DOT
git track board --graph       # the graph view
```

Edges point **blocker → blocked**, and ready nodes are marked. Paste the Mermaid
output into a GitHub comment, drop the DOT into Graphviz, or let the website
render it (below).

## On the website

The generated site and the live preview both include a board page:

```sh
git track web --out ./site    # ./site/board.html
git track serve               # /board
```

The board page shows the Ready and Blocked lists as plain, offline HTML — the
always-usable fallback — with the Mermaid dependency graph layered on top as a
progressive enhancement that draws when the page has network access. Offline or
on a dumb host, the lists still tell you everything; the graph is a bonus.

## Reading the board as a roadmap

Because blocking is derived from `depends-on` and `child-of`, the board doubles
as a roadmap:

- The **Ready** column is your unblocked work frontier — start here.
- Each **Blocked** entry names what it waits on, so you can trace a chain back to
  the one task that unblocks several others.
- An **epic** (parent) appears blocked until its children close, so epics
  naturally sink to the bottom until their work is done — a rollup without a
  separate progress field to maintain.

## Keeping it accurate

The board is recomputed from authored data every time — there is no stored board
state to drift. If you ever edit the tracker by hand and want to be sure the
local cache matches, run:

```sh
git track fsck            # verify the derived cache
git track fsck --write    # rebuild it from the authoritative objects
```
