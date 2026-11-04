# Commit integration

gitrack ties the tracker to your code history: you can close and relate tracks
from a commit message, keep a fix and its close in one commit, and see which
commits reference a track — all without a server watching your pushes.

## Every change is its own commit

Every mutating command (`create`, `update`, `set`, `close`, `reopen`,
`comment`, the relationship verbs, `config status ...`) commits its change as
its own revision, using your git identity. The commit is **scoped to
`.gitrack/`**: it stages and commits only the tracker path, so anything you have
staged *outside* `.gitrack/` stays staged and is never swept into a tracker
commit. The git-ignored cache never enters a commit.

Two flags apply to every mutating command:

- `--include <path>` (repeatable) adds a related file to the committed revision,
  so you can close a bug **in the same commit as its fix**.
- `--no-commit` leaves the change staged without committing, so you can batch
  several tracker changes and your own edits into one manual commit.

```sh
# close a bug together with the fix, in one commit
git track close token-leak --include src/log.go
```

## Closing tracks from a commit message

`git track commit` processes the **commit message's own directives** and bundles
the result as one revision:

```sh
git track commit -m "Add the login form

Closes #new-login-flow"
```

gitrack obtains the message (from `-m`, or `$EDITOR`), parses every
`<verb> #<handle>` in it, applies the closing directives, stages the tracker
changes, and commits. With `-- <file>...` the commit is scoped to those files
plus `.gitrack/`; with no pathspec it commits the staged index.

```sh
git track commit -m "Fix token redaction, Closes #token-leak" -- src/log.go
```

Shortcuts:

- A positional id — `git track commit token-leak -m "..."` — injects a
  `closes #<slug>` directive for you.
- `--status <s>` sets the closing status (default `done`).

## Closing keywords and lists

Directives use the GitHub + GitLab union of keywords, case-insensitive so
Conventional-Commit footers are recognized:

- **closing**: `close`/`closes`/`closed`/`closing`, `fix`/`fixes`/`fixed`/
  `fixing`, `resolve`/`resolves`/`resolved`/`resolving`,
  `implement`/`implements`/`implemented`/`implementing`
- **relates** (informational): `relate`/`relates`/`related`
  (`relates to`, `related to`)

A keyword governs the **whole list** of refs that follows it (the GitLab
behavior):

```
Closes #a #b #c
Resolves #a, #b, #c
Closes #a, #b, and #c
fixes #d, resolve #e
```

All of these attribute the keyword to every listed track. A bare `#<id>` with no
keyword is a **mention**, not a close.

## No passive auto-close

Observing a `closes #...` commit never auto-closes a track. gitrack has no
server to watch your pushes, so a state change happens only through
`git track close` or an explicit `git track commit` that you run. This keeps
closing deliberate and local.

## Mentions and referenced-by

A track's **mentions** are derived from git history (never stored): commits
whose message references the track via `#<handle>`, classified by the governing
keyword. `git track show` and the track's web page list them (short sha + verb +
subject), so the source tree cross-references the tracker.

A track's **referenced-by** is likewise derived — the union of commits that
touched the track's directory and code commits that opted in with a
`Gitrack-Ref: <slug>` trailer. Nothing is stored; both are recomputed from
`git log`.
