# ADR-0006: Kinds are an open, minimal, cosmetic set — `task` is the default

## Status
accepted

## Context
The stakeholder wants the simplest possible set of kinds, ideally simple enough
that gitrack can replace a plain TODO list. Earlier framing already held that a
track's kind is "generally irrelevant" to storage (ADR-0002, ADR-0003): a
project, epic, issue, bug, and note are the same object with different labels.

A TODO list is just tracks with a `state`. If adopting gitrack for a TODO costs
the user a taxonomy decision up front, it is too heavy.

## Decision
We will treat `kind` as a **cosmetic, open-ended label with no storage or
scheduling meaning**, defaulting to `task`. The only kind that carries extra
*behavior* is containment via the `child-of` relationship (ADR-0005), and that
behavior is triggered by the relationship, not by the kind. `.gitrack/config/kinds`
is an optional allow-list; when absent, any kind string is accepted and `task`
is used when none is given.

The shipped default `kinds` list is deliberately short:
`task`, `bug`, `note`. `epic` and `project` are offered as conventions for
"a track other tracks are `child-of`," but they are not special-cased in code —
an `epic` is just a `task` that happens to have children.

## Consequences
- Easier: `gitrack new "buy milk"` with no kind creates a `task`; gitrack is a
  usable TODO list with zero configuration.
- Easier: no code branches on kind; the graph and rollup logic key on
  relationships only. Maximum "dumb code."
- Harder: kinds carry no guarantees, so a typo'd kind (`buggg`) is silently
  accepted unless the user opts into a `kinds` allow-list. We accept this; the
  allow-list is there for teams that want rigor.
- Cost accepted: "epic"/"project" being conventions rather than types means
  tooling must infer hierarchy from `child-of`, not from kind. Already true.

## Verification
- [ ] Creating a track with no kind yields `kind` = `task`.
- [ ] With no `.gitrack/config/kinds`, an arbitrary kind string is accepted.
- [ ] With a `kinds` allow-list, a kind outside it is rejected by the CLI.
- [ ] No graph/rollup/scheduling code reads `kind`; it reads relationships only.
