# ADR-0023: Comments are UUID-named Markdown; author/date derived from git

## Status
accepted

## Context
The original comment format (FORMAT.md §5, grounded by ADR-0008) stored each
comment as `comments/<id>.md` with YAML front matter carrying `id`, `author`,
`created`, and `reply-to`. Two problems, both stakeholder-flagged:

1. **Comment id must be collision-safe.** Two users commenting offline on
   different branches can mint the same id and collide on merge — the exact risk
   that made track ids UUIDs (ADR-0002). A content SHA was considered and
   rejected: it dedupes, but editing a comment changes its hash, so the id is not
   stable and any `reply-to` referencing it breaks.
2. **author/created duplicate git.** git already records who authored, and when,
   the commit that added `comments/<id>.md`. Storing them in front matter
   duplicates git and can drift (rebase, amend, hand-edit) — the same reason
   `referenced-by` is derived, not stored (ADR-0009). Verified: `git log` on the
   comment file path recovers author (`%an <%ae>`) and created date (`%aI` of
   the add-commit).

## Decision
A comment is a Markdown file `comments/<uuid>.md` where `<uuid>` is a **UUIDv7**
minted at creation (reusing the track id generator; stable across edits,
collision-free offline). The file is **pure Markdown body** with no author and
no date.

The only metadata a comment may carry is **`reply-to`** (the parent comment's
UUID, for threading), because that is authored relationship data git cannot
derive. When present it is a minimal YAML front-matter block containing only
`reply-to`; a top-level comment has no front matter at all — just prose.

Author and created-date are **derived** from git history of the comment's file
path:

- author = `git log -1 --format='%an <%ae>' -- comments/<uuid>.md`
- created = `git log --diff-filter=A --format='%aI' -- comments/<uuid>.md`

This refines ADR-0008 (comment front matter) and extends ADR-0009's
derive-don't-duplicate rule to comment authorship.

## Consequences
- Easier: comments never collide across branches; the id is stable across edits.
- Easier: a comment is "just Markdown" — readable and writable with no tooling,
  and authorship cannot drift from git truth.
- Easier: `get <ref>.comments` lists bare UUIDs (ADR-0022); each is a real,
  typeable-enough handle.
- Harder: showing a comment's author/date requires a `git log` call (derived,
  cacheable), not a file read. Accepted — same tradeoff as `referenced-by`.
- Harder: `reply-to` is the one bit of front matter; a parser must tolerate both
  "front matter + body" and "body only". Small.
- Cost accepted: a shallow clone without history cannot show author/date; it
  degrades to showing the body (and the committing identity if available).

## Verification
- [ ] A new comment is `comments/<uuidv7>.md`; the id is stable across edits.
- [ ] A top-level comment file is pure Markdown with no front matter.
- [ ] A reply carries only `reply-to: <uuid>` in front matter, nothing else.
- [ ] No comment stores `author` or `created`; both are read from `git log` on
      the comment file path.
- [ ] Two comments created on separate branches have distinct ids after merge.
