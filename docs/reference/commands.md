# Command reference

Every gitrack command is a verb, run as `git track <command>` (or equivalently
`git-track <command>`). This page lists them all; the guide pages cover each
subject in depth.

A **reference** (`<ref>`) is a track's full UUID, short UUID, or slug — see
[References](../guide/references.md).

## Lifecycle

### `init [--no-commit]`
Initialize a tracker in the current repository, creating `.gitrack/` and
committing the scaffold. `--no-commit` leaves it staged.

### `create [-k kind] <title>`
Create a track with `state=open`, `status=new`, and a slug derived from the
title. Prints the new UUID. `-k/--kind` sets the cosmetic kind (default `task`).

### `update <ref> [flags]`
Change a track's fields. Flags: `-t/--title`, `-k/--kind`, `-d/--description`,
`-s/--status`. The `-s` status move is validated against the workflow;
`state` is not an `update` flag.

### `close <ref> [--status=s]`
Close a track: `state=closed`, `status=done` (or `--status=<s>`).

### `reopen <ref>`
Reopen a track: `state=open`, `status=reopened`.

## Reading

### `show <ref>`
Display a track: its fields, computed blocked/unblocked status, and mentions.

### `get [--json] <ref>[.path]`
Read a field, list a multi-value attribute's bare ids, or dump the whole object.
Examples: `get <ref>.status`, `get <ref>.comments`, `get <ref>.comments.<id>`,
`get --json <ref>`. See [Working with tracks](../guide/tracks.md).

### `set <ref>.<key> <value>`
Write a free-form field (`title`, `description`, `kind`, ...). Refuses the
governed fields `state`/`status` — use `close`/`reopen` and `update -s`.

### `list [--state s] [-s status] [-k kind]`
List tracks, optionally filtered by state, status, or kind. (`ls` is an alias.)

### `search <query>`
Find tracks by text in their title, description, or slug.

## Relationships

### `depends-on <ref> <target>`
Mark `<ref>` blocked by `<target>` (ref waits for target).

### `child-of <ref> <parent>`
Place `<ref>` under `<parent>` (single parent; the parent waits for the child).

### `related <ref> <target>`
A soft link with no scheduling effect.

### `duplicate-of <ref> <target>`
Mark `<ref>` a duplicate of `<target>`.

### `unlink <ref> <rel> <target>`
Remove a relationship edge, e.g. `unlink a depends-on b`.

See [Relationships](../guide/relationships.md).

## Comments

### `comment <ref> [-m msg] [--reply-to=<cid>]`
Add a Markdown comment. With no `-m`, `$EDITOR` opens. `--reply-to` threads under
a parent comment. Author and date come from git. See
[Comments](../guide/comments.md).

## Commit integration

### `commit [<id>...] [-m msg] [--status s] [-- <file>...]`
Commit, applying the message's `closes`/`relates` `#ref` directives. A positional
`<id>` injects a `closes #<slug>` directive; `--status` sets the closing status;
`-- <file>...` scopes the commit to those files plus `.gitrack/`. See
[Commit integration](../guide/commit-integration.md).

**Shared flags on every mutating command:** `--include <path>` (repeatable)
bundles a related file into the committed revision; `--no-commit` leaves the
change staged.

## Configuration

### `config status ...`
Edit status workflows:

- `config status add <name> [--next a,b,c]` — create a status with transitions.
- `config status set-next <name> [next...]` — replace its transitions.
- `config status list` — show all statuses and transitions.
- `config status rm <name>` — remove a user-defined status.

See [Status workflows](../guide/workflow.md) and
[Configuration](../guide/configuration.md).

## Views and output

### `board [--graph|--mermaid|--dot]`
Show Ready/Blocked work, or emit the dependency digraph (Mermaid or DOT). See
[Boards and roadmaps](../guide/boards.md).

### `web --out <dir>`
Generate a self-contained, offline-capable static site. See
[The project website](../guide/website.md).

### `serve [--addr 127.0.0.1:0]`
Start a loopback-only live preview server on a random port (or the given
`--addr`), rendering the current tracker on each request.

## Maintenance

### `fsck [--write]`
Verify the derived cache (and report divergence from config guardrails);
`--write` rebuilds the cache from the authoritative objects.

## Help

### `--help` / `help`
Print usage. `git track help --man` emits the roff man page (same source as
`--help`, so they never drift).
