# ADR-0016: Ship as `git-track` on PATH, with a `git-track(1)` man page

## Status
accepted

## Context
ADR-0015 chose a single Go binary invoked as `git track`. The exact integration
mechanism, and one sharp edge, need to be pinned because they drive the binary
name and the docs we ship.

git's subcommand dispatch is a thin wrapper: `git <foo>` runs an executable
named `git-<foo>` found in the git exec path (`/usr/lib/git-core`) or anywhere
on `$PATH`. So a binary named `git-track` on `PATH` makes `git track ...` work
with no registration at all.

The one hiccup (stakeholder-flagged, verified): `git help track` and
`git track --help` are **intentionally** routed through the man system — git
does this so help has a single source and cannot drift from the code. They look
for a man page `git-track(1)`. Without it, `git track` runs fine but
`git help track` fails. The subcommand executable's own `--help` handling does
not satisfy `git`'s man-page lookup.

## Decision
We will name the built binary **`git-track`** (and install it on `PATH`), so
`git track <args>` works by git's native dispatch with zero configuration. The
Go module builds this binary from `./cmd/git-track`. We also **ship a
`git-track(1)` man page** and install it where `man` resolves it (a `man1`
directory on `MANPATH`, e.g. `$PREFIX/share/man/man1/git-track.1`), so
`git help track` and `git track --help` work like any built-in.

The man page is generated from a single source kept in the repo (so CLI help and
the man page cannot drift — the same reason git uses man pages), and the release
packaging installs both the binary and the man page. A convenience alias
`gitrack` may symlink to `git-track` for direct invocation, but `git-track` is
the canonical name.

## Consequences
- Easier: `git track` works the instant the binary is on `PATH`; no git config,
  no alias, no registration.
- Easier: `git help track` / `git track --help` behave like a built-in once the
  man page is installed, giving a first-class feel.
- Harder: we must build and install a man page, and keep it generated from one
  source to avoid help drift. We accept this; it is a standard CLI release task
  and git-bug-class tools do it routinely.
- Harder: on a bare binary drop (no installer, man page absent), `git track`
  works but `git help track` does not. We document this as the known limitation
  and have `git track --help` print usage directly as a fallback when the man
  page is missing, so the CLI is never help-less.
- Cost accepted: release artifacts carry a man page per platform packaging.

## Verification
- [ ] The build produces an executable named `git-track`; placing it on `$PATH`
      makes `git track ...` work with no further setup.
- [ ] A `git-track(1)` man page is generated from a single in-repo source and
      installed to a `man1` dir on `MANPATH`; `git help track` opens it.
- [ ] `git track --help` prints usage even when the man page is absent (fallback).
- [ ] CLI help text and the man page derive from one source (no drift).
