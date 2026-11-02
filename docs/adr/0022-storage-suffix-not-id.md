# ADR-0022: Storage suffixes (e.g. `.md`) are not part of the addressable id

## Status
accepted

## Context
ADR-0021 added dotted-path accessors and let `comments.<id>` resolve the file
`comments/<id>.md`. The first implementation matched a bare segment by stripping
*any* single extension from *any* entry — too loose: it would match `foo.txt`
for `foo`, and it treated the extension as an accident rather than a declared
property.

The stakeholder clarified the intent: `.md` is a **storage detail** marking the
file as Markdown (the comment body). It is the suffix gitrack adds when it
creates/reads/writes the entry, and it is **not** part of the id, the command,
or how the entry is addressed.

## Decision
A multi-value attribute may declare a **storage suffix** for its entries. The
suffix is a property of the attribute, known to the code, never a wildcard:

- `comments` entries are stored as `<id>.md` (Markdown body). The addressable id
  is the bare `<id>`; gitrack appends `.md` on write and resolves `<id>` to
  `<id>.md` on read.
- Attributes with no declared suffix (`labels`, `assignees`, the relationship
  dirs) store bare-named entries and address exactly.

Accessor behavior (refining ADR-0021):

- `get <ref>.comments` lists **bare ids** (the `.md` suffix is stripped from the
  listing), so the listed values are exactly what you pass back in.
- `get <ref>.comments.<id>` resolves `comments/<id>.md`. Resolution tries the
  exact name first, then the name plus the attribute's declared suffix — it
  never strips an arbitrary extension.
- A path segment that matches neither is an error.

## Consequences
- Easier: ids are clean and round-trip — `get .comments` yields values you can
  feed straight back to `get .comments.<id>`.
- Easier: the Markdown-ness of comments is a storage fact in one place, not
  leaked into the CLI vocabulary.
- Harder: the code needs a small attribute→suffix table. Trivial and explicit;
  far safer than the arbitrary-extension strip it replaces.
- Cost accepted: adding a future attribute with its own file type means adding
  one table entry.

## Verification
- [ ] `get <ref>.comments` lists bare ids with no `.md`.
- [ ] `get <ref>.comments.<id>` reads `comments/<id>.md`.
- [ ] A bare segment is resolved only via an exact name or the attribute's
      declared suffix, never by stripping an arbitrary extension (e.g. a
      `notes.txt` entry is not matched by segment `notes` unless `.txt` is the
      attribute's declared suffix).
