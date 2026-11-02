# gitrack

A distributed Issue / Task / Note / Epic / Project tracker whose state lives
**in your git working tree** and merges as part of an ordinary `git merge`.

Fix a bug on a topic branch, and merging that branch brings the code change and
the tracker update together — one history, one operation. No separate service,
no separate ref namespace, fully usable offline.

The goal is a tracker with **no hosted service** — sync through any git remote
(shared drive, S3 bucket, drop-box, ssh, IPFS) and serve the views as **static
content** from a dumb host or straight off disk. No server to own, patch, or
trust; a far smaller attack surface; and less energy than an always-on hosted
app. See [`docs/VISION.md`](docs/VISION.md) and ADR-0029.

This is a net-new project inspired by the ideas in the original shell-era
gitrack — now archived at
[major0/gitrack-old](https://github.com/major0/gitrack-old) — and by the
broader family of VCS-embedded trackers (git-bug, BugsEverywhere, Ditz,
driusan/bug). The
**on-disk format is the product** (see `docs/FORMAT.md`); the CLI is one
consumer of it. The implementation is **Go**, distributed as a single binary
and invoked as a `git track` subcommand (ADR-0015).

## Status

Early. The design foundations and on-disk format are specified (`docs/`), and
the Go CLI implements `init`, `config`, `create`, `update`, `get`, `set`,
`show`, `list`, `search`, `close`, `reopen`, `comment`, `serve` (loopback live
preview), and `fsck`.

Install the binary and man page with `scripts/install.sh` (defaults to
`~/.local`); then `git track` works as a subcommand and `git help track` opens
the man page. Or just `go build -o git-track ./cmd/git-track` for the binary
alone. The man page is generated from the CLI's own command data
(`git track help --man`), so it never drifts from `--help`.

## Core ideas

- **One universal object — a "track."** project, epic, issue, bug, task, and
  note are the same object distinguished by a `kind` field. Each has a stable
  UUID; the kind is mostly irrelevant to storage.
- **State lives with the code.** Data is plaintext in `.gitrack/` in the working
  tree, so it branches and merges with the code as one `git merge`.
- **Smart data, dumb code.** The layout *is* the database: a single-valued field
  is a file whose name is the key and content is the value
  (`.gitrack/<id>/state`); a multi-valued field is a directory
  (`comments/`, `depends-on/`). Reading a field is `cat`. There is no index to
  trust, no state to recompute.
- **Conflicts are a feature.** Because `state` is one atomic file, two people
  setting it differently on different branches produce a real git conflict for a
  human to resolve — while independent field edits and multi-value appends merge
  silently. This is proven empirically in `tests/merge-semantics.sh`.

## Planned surface

- A Go CLI shipped as `git-track` on `PATH`, invoked as `git track` by git's
  native subcommand dispatch (ADR-0016), for creating and updating tracks and
  viewing the tracker.
- A statically rendered web view and issues view — self-contained output that
  deploys to any dumb host (S3, shared drive, Pages, IPFS) or opens as `file://`,
  with no runtime server (ADR-0029).
- GitHub-style autolinked references across the web and issue views.
- Project boards and roadmaps with a digraph of *unblocked work* — already built:
  `git track board` (text, `--mermaid`, `--dot`) and the `/board` page in
  `git track serve` (local preview).

## Documentation

- [`docs/VISION.md`](docs/VISION.md) — the why: a tracker with no hosted service.
- [`docs/FORMAT.md`](docs/FORMAT.md) — the authoritative on-disk format spec.
- [`docs/PRIOR_ART.md`](docs/PRIOR_ART.md) — survey of prior trackers and the
  lessons taken from each.
- [`docs/adr/`](docs/adr/) — architecture decision records.

## License

[MIT](LICENSE). (The original shell-era gitrack was GPLv2; this net-new
reimplementation is MIT — permissive, close to public domain, and the safer
choice for agent-assisted development under current copyright law.)
