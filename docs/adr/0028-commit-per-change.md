# ADR-0028: Every mutating command commits its own scoped changeset

## Status
accepted

## Context
ADR-0016-era behavior and the previous turn left only `init` auto-committing;
other mutating commands wrote working-tree files for the user to stage. The
stakeholder now wants every command that changes a track to commit that change
as its own revision — but with two guarantees:

1. the commit must be **scoped to `.gitrack/`** and must NOT accidentally pick up
   unrelated changes the user has staged elsewhere; and
2. a related file may be **deliberately bundled** via `--include <path>`.

This supersedes the "only init commits" stance (prior turn). The merges-with-code
property is preserved — and made explicit — by `--include`: you bundle the fix
with the tracker change on purpose, rather than relying on the whole index.

We verified the mechanism (`tmp/probe-pathspec-commit.sh`): `git add -- <paths>`
followed by `git commit -- <paths>` commits only the matching paths and leaves an
unrelated staged file out of the commit (it stays staged). So a pathspec-scoped
commit gives the required isolation.

## Decision
Every mutating command (`create`, `update`, `set`, `close`, `reopen`, `comment`,
`depends-on`/`child-of`/`related`/`duplicate-of`, `unlink`, `config status
add`/`set-next`/`rm`) commits its change as a single revision, scoped to the
tracker path:

- The commit stages and commits **only** `.gitrack/` (plus any `--include`
  paths), using `git add -- <paths>` then `git commit -- <paths>`. Changes the
  user has staged outside those paths are never swept in and remain staged.
- `--include <path>` (repeatable) adds a path to the committed changeset, so a
  related code file rides in the same revision as the tracker change. This is the
  explicit, opt-in form of the merges-with-code property.
- Commits use the user's own git identity, config, and hooks (gitrack never
  injects an identity). The git-ignored cache (`.gitrack/cache/`) is never part
  of a commit.
- `init` keeps `--no-commit`; the other commands gain `--no-commit` too, so any
  single operation can opt out and leave its changes staged.
- Each command writes a descriptive commit subject (e.g. `gitrack: create <slug>`,
  `gitrack: close <slug>`).

The derived cache is rebuilt after the write but BEFORE the commit scope is
computed; since the cache is git-ignored it never enters the commit regardless.

## Consequences
- Easier: a clean, per-change history — each tracker action is one reviewable,
  revertable revision, and `git log -- .gitrack/<id>/` reads as that track's
  history (which also feeds the derived author/date, ADR-0009).
- Easier: `--include` makes "close the bug in the commit that fixes it" explicit
  and intentional, not an accident of what happened to be staged.
- Safer: pathspec scoping guarantees unrelated staged work is never captured.
- Harder: more commits (one per action). Acceptable; it matches git-bug-class
  tools and the stakeholder's "each change is its own revision" intent.
- Harder: a command now performs a git write; `--no-commit` is the escape for
  batching several changes into one manual commit.
- Cost accepted: a pre-existing merge/rebase in progress could make an
  auto-commit surprising; the command reports the git error and `--no-commit`
  sidesteps it.

## Verification
- [ ] `git track create` (and update/set/close/reopen/comment/relationship/
      config) each produce exactly one commit scoped to `.gitrack/` (+ includes).
- [ ] A file staged outside `.gitrack/` before the command is NOT in the
      resulting commit and remains staged.
- [ ] `--include <path>` adds that path to the committed changeset; repeatable.
- [ ] `--no-commit` on any mutating command leaves its changes staged, no commit.
- [ ] The git-ignored cache never appears in any commit.
- [ ] Commits use the user's git identity.
