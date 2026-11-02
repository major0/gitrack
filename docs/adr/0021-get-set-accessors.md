# ADR-0021: Dotted-path `get`/`set` accessors alongside invariant-aware `update`

## Status
accepted

## Context
The command surface is growing (`init`, `config`, `list`, `search`, `create`,
`close`, `reopen`, `update`, `show`, `comment`, ...). The stakeholder is rightly
concerned that `update` becomes an overloaded catch-all, and proposes
script/user-friendly accessors:

- `get [--json] <object>[.<key>]`
- `set <object>.<key> <value>`

with path descent into multi-value attributes: `get <object>.comments`,
`get <object>.comments.<id>`.

This maps directly onto the format (ADR-0003): a field is a file, a multi-value
attribute is a directory, so `<object>.<key>` is a path *into the tree*. Slugs
are `[a-z0-9-]` and UUIDs are hex+`-` (no `.`), so the first `.` unambiguously
separates the object reference from the key path, and further `.` segments
descend into the tree.

The risk: a blind `set` that writes any field would bypass the lifecycle
invariants of ADR-0019 — `state` is changed only by `close`/`reopen`, and
`status` moves are validated against the workflow. `set object.state closed`
could create the incoherent `state=closed, status=new` pair we designed out.

## Decision
We add two low-level, scriptable accessors and KEEP `update` as the high-level,
invariant-aware editor. They are complementary, not redundant:

- **`get [--json] <ref>[.<path>]`** — read-only, universal. Resolves `<ref>`
  (uuid/short/slug), then descends `<path>` through the tree: a field path reads
  the file; a directory path lists entries; a deeper path reads a nested file
  (e.g. `comments.<id>`). With no `.<path>`, returns the whole object. `--json`
  emits machine-readable output; without it, human-readable. No invariant risk.

- **`set <ref>.<key> <value>`** — writes **free-form** fields only (`title`,
  `description`, `kind`, and multi-value adds under `labels`/`assignees`). For
  the **governed** fields it refuses and redirects: `set <ref>.state ...` and
  `set <ref>.status ...` error with "use close/reopen to change state; update
  --status to move status", so the lifecycle has exactly one validated path.

- **`update <ref> [flags]`** — unchanged in role: the human-friendly multi-field
  editor that applies validation (status transitions via ADR-0019). Low-level
  field pokes move to `get`/`set`, so `update` stays focused and does not become
  the catch-all.

Path grammar: split `<ref>.<path>` on the FIRST `.`; the remainder is a
dot-separated descent (`comments`, then `comments.<id>`). A path that does not
resolve to a file or directory is an error.

## Consequences
- Easier: scripts get a clean `get --json`/`set` pair that mirrors the on-disk
  tree one-to-one — the smart-data layout surfaced as the CLI.
- Easier: `update` stops accreting; each field poke has a low-level home.
- Easier: the lifecycle keeps a single validated path; `set` cannot corrupt it.
- Harder: three commands touch fields (`get`/`set`/`update`); we document that
  `get`/`set` are low-level and `update`/`close`/`reopen` are the safe verbs.
- Cost accepted: `set` must know which fields are governed. Small, explicit list.

## Verification
- [ ] `get <ref>` returns the whole object; `get <ref>.state` reads one field;
      `get <ref>.comments` lists the dir; `get <ref>.comments.<id>` reads one.
- [ ] `get --json <ref>[.path]` emits valid JSON.
- [ ] `set <ref>.title "X"` writes the field; `set <ref>.state closed` and
      `set <ref>.status X` are refused with a redirect to the proper verb.
- [ ] `update` remains and still validates status transitions.
- [ ] The first `.` splits ref from path; slugs/uuids never contain `.`.
