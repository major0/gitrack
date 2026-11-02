# ADR-0015: Implement gitrack in Go

## Status
accepted

## Context
gitrack must be reimplemented in a language more suitable than the original
POSIX shell. The candidates are Go and Python. The choice is hard to reverse
once code exists, so it is decided deliberately against what the tool actually
does.

What gitrack's core workload is, given the format we designed (ADR-0001..0014):

- A **filesystem-walk-heavy, many-tiny-files** engine. The derived cache
  (`.gitrack/cache/`, ADR-0014) and `gitrack fsck` stat/readdir/symlink across
  one directory per track and one file per field/edge. This is syscall-bound.
- Constant interaction with **git** (resolve refs and short-ids, `git log` for
  `referenced-by`, detect merges) — either by shelling out or via a native
  library.
- A **CLI run very often**, as a `git track ...` subcommand on developer
  machines and in CI, plus a static-site generator for GitHub Pages.

Ecosystem facts (verified):

- **git-bug**, the most mature prior art and our primary design reference, is
  written in Go and distributed as a **single binary** across platforms — the
  natural distribution model for a `git` subcommand.
- **go-git** is a pure-Go git implementation (plumbing + porcelain), so ref
  resolution and history reads need no C dependency; shelling to `git` remains
  an option.
- The original "reference implementation" goal is satisfied by the
  **language-agnostic format spec** (`docs/FORMAT.md`) plus conformance tests
  (`tests/`), not by the implementation language. The spec is the reference;
  the code is one conforming consumer.

## Decision
We will implement gitrack in **Go**, distributed as a single static binary
installed on `PATH` and invoked as `git track` / `gitrack`. The static-site
generator uses Go's stdlib `html/template`. git interaction is via shelling to
the system `git` initially (simplest, matches user's real git config and
credentials), with `go-git` available where an in-process library is clearly
better; the choice is an internal detail behind a small git-access interface.

## Consequences
- Easier: one static binary, no interpreter or venv — the right friction level
  for a tool every developer and CI job runs. Instant startup per invocation.
- Easier: fast, goroutine-parallel directory walks for the cache and `fsck`,
  which is the performance-sensitive core the stakeholder flagged.
- Easier: fits the git/git-bug/go-git ecosystem gitrack lives in.
- Easier: `html/template` covers static-site generation without extra deps.
- Harder: Go is less casually hackable than Python for drive-by contributors.
  We accept this; the format spec + conformance tests keep the project
  approachable and permit an alternate-language implementation later.
- Harder: Go's templating and Markdown story needs a GFM library
  (e.g. yuin/goldmark) for task-list rendering (ADR-0008); a small, well-
  maintained dependency, pinned.
- Cost accepted: cross-compilation and release packaging for multiple platforms
  is our responsibility (git-bug shows this is routine for Go CLIs).

## Verification
- [ ] The repository builds a single `gitrack` binary with `go build ./...`.
- [ ] `gitrack` runs as a git subcommand (`git track ...`) with no runtime
      dependency beyond the binary and `git`.
- [ ] The cache build and `fsck` walk the tree without shelling out per file.
- [ ] Static-site generation uses `html/template` + a pinned GFM Markdown lib.
- [ ] `docs/FORMAT.md` + `tests/` remain the language-agnostic reference; the
      merge-semantics test runs independent of the Go code.
