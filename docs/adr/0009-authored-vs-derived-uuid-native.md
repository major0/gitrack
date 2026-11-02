# ADR-0009: Authored vs derived data, with UUID-native reference derivation

## Status
accepted

## Context
ADR-0007 drew the right authored/derived line and correctly forbade
write-on-write. But its mechanism for deriving `referenced-by` was wrong: it
proposed `git log --grep=gitrack:<uuid>`, which assumes a human types the
track's UUID into a commit message. Our tracks are identified by **UUIDs, not
git refs** — nobody will paste `0192f1a2-7c3e-7abc-def0-112233445566` into a
commit subject. git gets away with `--grep` of short hashes and GitHub with
`#123` precisely because those are short and typeable; a UUID is neither.

This ADR keeps ADR-0007's authored/derived split and its no-write-on-write rule
verbatim, and replaces only the *reference-derivation mechanism* with one that
never requires typing a UUID. Verified in `tmp/ref-scan.sh`.

## Decision
The authored vs derived split is unchanged from ADR-0007:

- **Authored** (stored, conflicts on divergent edit): `kind`, `title`,
  `description`, `state`, `slug`, `labels/`, `assignees/`, `comments/`,
  `attachments/`, and relationship link dirs (`depends-on/`, `child-of/`,
  `related/`, `duplicate-of/`).
- **Derived** (computed on demand, never stored as truth): `referenced-by`,
  blocked/unblocked, rollup/progress, inverse edges, and the rendered
  checked-state of dependency-backed task-list items.
- No gitrack operation writes derived data back into an authored file. A `state`
  change edits exactly one file; everything "automatic" happens at render/query
  time against live authored state.

**`referenced-by` is derived from two UUID-native signals, unioned — no UUID is
ever typed by a human:**

1. **Structural** — commits that touch the track's directory:
   `git log -- .gitrack/<uuid>/`. Every create, state change, comment, and
   relationship edit of a track lives under its dir, so this captures all
   tracker activity and any code commit that edited the track in the same
   commit. It cannot drift and needs no convention.
2. **Explicit opt-in** — a code commit that references a track it did *not*
   touch adds a trailer `Gitrack-Ref: <handle>`, where `<handle>` is a
   **typeable** slug (the track's `slug` file) or an unambiguous short-id prefix
   (resolved like a git short hash), never the full UUID. Derived via
   `git log --grep='Gitrack-Ref: <handle>'`, with the handle resolved to the
   UUID through the track's `slug`.

Each track therefore carries an authored, human-chosen `slug` (unique within the
repo, kebab-case, derived from the title at creation, editable). The slug is the
typeable handle for references and URLs; the UUID remains the canonical identity.

## Consequences
- Easier: `referenced-by` is always accurate and costs no stored state; closing
  or committing never mutates another track.
- Easier: humans reference tracks by a readable slug, not a UUID — the ergonomics
  git/GitHub have, on a UUID-keyed store.
- Easier: structural derivation means even tracker edits made with no tool (a
  hand `cat > state`) still show up as references.
- Harder: slugs must be unique and kept resolvable; a retitle may change the
  preferred slug, so old slugs are kept as aliases (a `slug.aliases/` multi-value
  dir) rather than broken. The CLI owns slug minting and alias capture.
- Harder: the renderer unions two queries per track for `referenced-by`. Pure and
  cacheable under the git-ignored `.gitrack/index/`.
- Cost accepted: a short-id prefix can become ambiguous as tracks grow; the CLI
  resolves and errors on ambiguity exactly as git does with short hashes.

## Verification
- [ ] Closing a track modifies only that track's `state` file; no other track's
      files change on disk.
- [ ] `referenced-by` for a track is the union of `git log -- .gitrack/<uuid>/`
      and `git log --grep='Gitrack-Ref: <handle>'`, with no stored
      `referenced-by/` directory as source of truth, and no full UUID required in
      any commit message.
- [ ] Each track has an authored, unique `slug`; references and URLs use the slug
      or a short-id prefix, resolved to the UUID.
- [ ] A retitle preserves old slugs as resolvable aliases.
- [ ] Blocked/unblocked, rollup, inverse edges, and dependency-backed checkbox
      state are computed, not stored; any cache lives under git-ignored
      `.gitrack/index/`.
