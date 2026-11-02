# ADR-0008: GFM task lists in descriptions; dependency-backed items render from state

## Status
accepted

## Context
The stakeholder wants GitHub-style task lists in a track's `description`, able to
be marked off, and specifically: when a track that is a *dependency* of another
is closed, the depending track's task-list should reflect that automatically.

GitHub Flavored Markdown task lists are `- [ ]` (incomplete) and `- [x]`
(complete), whitespace-strict, and GFM already allows an item to contain a
reference like `#1234`. We must match GFM so our rendered pages and a plain
GitHub preview agree.

ADR-0007 forbids writing a checkbox back into the authored `description` as a
side effect of closing another track. So a dependency-backed box cannot be a
stored `[x]`; its checked-ness must be *derived* from the dependency's state at
render time.

## Decision
We support two kinds of task-list item in `description`, both valid GFM:

1. **Free item** — an ordinary hand-authored checkbox: `- [ ] write the docs`.
   Its checked state is authored prose; the user toggles it by editing the
   description. gitrack never touches it.

2. **Dependency-backed item** — a checkbox whose text contains a track
   reference: `- [ ] gitrack:<handle> short label`, where `<handle>` is a
   typeable slug or short-id prefix (NOT a raw UUID — see ADR-0009). Its checked
   state is **derived**: the renderer resolves the handle to a UUID, reads that
   track's `state`, and shows `[x]` iff it is closed (state ∈
   `config/states.closed`), `[ ]` otherwise. The authored bytes are left as
   `- [ ]`; the *rendered* page shows the computed box. On create/edit,
   referencing a track in a description item also materializes a
   `depends-on/<uuid>` link (the item *is* the dependency), so the graph and the
   checklist share one source of truth.

Thus "auto-check when a dependency closes" happens with zero writes: closing the
dependency changes only its own `state` file (ADR-0007/0009); every page that
renders a dependency-backed item recomputes the box from that live state.

A reference uses a resolvable `<handle>` (slug or unambiguous short-id prefix,
resolved like git short hashes). Short human display uses the target's `title`.

## Consequences
- Easier: the checklist and the dependency graph cannot disagree — they are the
  same `depends-on` data viewed two ways.
- Easier: closing work anywhere updates every dependent checklist on next render,
  with no cascading writes and no merge hazard.
- Easier: a plain GitHub view still shows a normal (if statically `[ ]`) task
  list, so the format degrades gracefully without our renderer.
- Harder: a dependency-backed item's on-disk `[ ]` may not match its rendered
  `[x]`. We document that the *rendered* page is authoritative for derived boxes;
  the raw byte is just a placeholder. Free items have no such gap.
- Harder: editing a dependency-backed item's reference must stay in sync with the
  `depends-on/` link. The CLI owns this; a hand-edit that desyncs is flagged by a
  linter, not silently corrected.

## Verification
- [ ] A free `- [ ]`/`- [x]` item round-trips unchanged; gitrack never edits it.
- [ ] A `- [ ] gitrack:<id> ...` item renders `[x]` iff the target track's state
      is in `config/states.closed`, computed at render time.
- [ ] Creating/editing such an item materializes a matching `depends-on/<id>`
      link; removing it removes the link (CLI-enforced).
- [ ] Closing a dependency changes only its own `state` file, yet the dependent's
      rendered checklist shows the item checked.
- [ ] A linter flags a dependency-backed item whose `depends-on/` link is missing.
