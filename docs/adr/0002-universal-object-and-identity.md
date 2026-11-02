# ADR-0002: One universal object identified by a GUID

## Status
accepted

## Context
The original gitrack and most prior art special-case "types" (bug vs issue vs
note) with separate directories and sometimes separate schemas. The requirement
for this project is explicit: *the base object is the same regardless of whether
we are tracking a project, epic, issue, bug, task, or note.* It needs a GUID and
various metadata.

Identity choice matters because it is effectively irreversible once objects
exist. Candidates:

- **Sequence number** (`#1`, `#2`): collides across branches — two people each
  open "issue #5" on their own branch and the merge is ambiguous.
- **Path / content hash of the first event** (git-bug style, gitrack style):
  stable and branch-safe, but the identifier is tied to the first event's bytes
  and is awkward to mint offline before the event exists.
- **GUID (UUID)**: branch-safe, rename-safe, reclassification-safe, mintable
  offline with no coordination, and explicitly requested by the stakeholder.

## Decision
We will represent every tracked thing as a single **universal object** with a
`kind` field (`project | epic | issue | bug | task | note | ...`, open-ended and
config-driven), not as distinct per-type schemas. Each object is identified by a
**UUID (version 7)** minted at creation. The UUID — never a path, sequence
number, or kind — is the object's stable identity across branches, renames, and
reclassification between kinds.

UUIDv7 is chosen over v4 because it is time-ordered: listings and on-disk
sharding sort chronologically for free, while retaining v4's collision-free
offline minting.

## Consequences
- Easier: changing an object's `kind` (promote a note to a bug, a bug to an epic)
  is a metadata event, not a move — identity and history are preserved.
- Easier: two authors creating objects offline on different branches never
  collide; the merge is a clean union.
- Easier: every tool (CLI, web, issues, boards) keys on one stable id.
- Harder: a raw UUID is not human-friendly to type. We mitigate with
  git-style shortest-unique-prefix matching on the CLI (type as few hex chars as
  are unambiguous) and a human-readable `slug` derived from the title for display
  and URLs. The slug is cosmetic; the UUID is canonical.
- Cost accepted: UUIDs are longer on disk than sequence numbers. Negligible.

## Verification
- [ ] Creating any object mints a UUIDv7 and records `id` + `kind` in its first
      event.
- [ ] Changing `kind` is representable as an event without changing `id` or the
      on-disk directory identity.
- [ ] The CLI resolves an object by any unambiguous id prefix and errors with
      candidate list on ambiguity.
- [ ] Two objects created on separate branches with no network have distinct ids
      after merge (no collision).
