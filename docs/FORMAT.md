# gitrack on-disk format

This is the authoritative specification of the gitrack tracker database. It is
the contract. The CLI and every generator (web, issues, boards) read and write
*only* what this document defines. A conforming reimplementation in any language
that produces this layout is interoperable.

Grounding decisions:
- `docs/adr/0002-universal-object-and-identity.md` — one universal object, UUID.
- `docs/adr/0003-smart-data-dumb-code-layout.md` — filesystem-as-database,
  one file per field (supersedes the earlier event-log model of ADR-0001).
- `docs/adr/0005-relationship-direction.md` — one relationship primitive and its
  scheduling direction (supersedes ADR-0004).

## 1. Principles

1. **Smart data, dumb code.** The layout carries the structure. Reading a field
   is `cat`-ing a file; there is no database, no index to trust, no fold.
2. **Data lives in the working tree** so it merges with the code as one `git
   merge` operation.
3. **One universal object — a "track."** project, epic, issue, bug, task, note
   are all tracks distinguished by a `kind` field. The kind is mostly
   irrelevant to storage.
4. **Conflicts are a feature.** A single-value field is one file; two people
   setting it differently produces a git conflict for a human to resolve. This
   is intended (proven empirically — see ADR-0003).
5. **Multi-value is a directory.** Appends by different people are different
   files and merge as a clean union.
6. **Config is guardrails, not enforcement (ADR-0026).** The tree is openly
   editable (`cat`/`mkdir`/your editor) and merges happen outside the CLI, so
   `config/` (statuses, kinds, workflows) describes *intended* structure and
   keeps the CLI's well-behaved path consistent — it cannot constrain what the
   tree contains. `git track fsck` is the reconciler: it reports where
   authoritative data diverges from the guardrails as advisory warnings and
   never rewrites an authored field to match config.

### Committing

Every mutating command (`init`, `create`, `update`, `set`, `close`, `reopen`,
`comment`, the relationship verbs, `config status add/set-next/rm`) **commits
its change as its own revision** (ADR-0028), using your own git identity
(gitrack never injects one). The commit is **scoped to `.gitrack/`**: it stages
and commits only the tracker path via `git commit -- <paths>`, so changes you
have staged *outside* `.gitrack/` are never captured and stay staged. The
git-ignored cache (`.gitrack/cache/`) never enters a commit.

Two flags on every mutating command:
- `--include <path>` (repeatable) adds a related file to the committed revision,
  so you can close a bug *in the same commit as its fix* — the merges-with-code
  property (Principle 2), made explicit and intentional.
- `--no-commit` leaves the change staged without committing, so you can batch
  several tracker changes (and your own edits) into one manual commit.

## 2. The key/value rule (the whole model in one sentence)

> A single-valued attribute is a **file** whose name is the key and whose
> content is the value. A multi-valued attribute is a **directory** whose
> entries are the values.

Everything below is an application of that one rule.

## 3. Directory layout

```
.gitrack/
├── .gitignore                   # contains "/cache/" (keeps the cache untracked)
├── config/                      # repo-level configuration (see §7) — AUTHORITATIVE
│   ├── kinds                    # allowed track kinds, one per line
│   ├── states                   # allowed states, one per line
│   └── workflow/<kind>          # optional per-kind workflow (see §7)
├── cache/                       # DERIVED, git-ignored, local, rebuildable (§4.1)
│   ├── state/<state>/<uuid>     #   index: tracks by state (open/closed)
│   ├── status/<status>/<uuid>   #   index: tracks by status (ADR-0019)
│   ├── type/<kind>/<uuid>       #   index: tracks by kind
│   └── refs-in/<uuid>/<rel>/<referrer-uuid>   # backref index
└── <uuid>/                      # one directory per track — AUTHORITATIVE
    ├── kind                     # cosmetic label, default "task" (ADR-0006)
    ├── slug                     # typeable handle, unique in repo (ADR-0009)
    ├── slug.aliases/            # prior slugs kept resolvable after a retitle
    │   └── <old-slug>
    ├── title                    # one-line summary
    ├── description              # free-form body (Markdown, may hold task lists §8)
    ├── state                    # e.g. "in-progress" (atomic; conflicts on clash)
    ├── created                  # RFC3339 timestamp, display only
    ├── labels/                  # multi-value -> directory
    │   └── <label>              # empty file; name is the label
    ├── assignees/
    │   └── <identity>           # empty file; name is the assignee
    ├── comments/
    │   └── <uuid>.md            # one comment per file, YAML front matter (§5)
    ├── attachments/
    │   ├── <uuid>               # stored blob
    │   └── <uuid>.meta          # front matter: original name, type, or a
    │                            #   pointer to an external path (§6)
    ├── depends-on/              # forward relationship dirs (§4), source of truth
    │   └── <target-uuid>
    ├── child-of/
    │   └── <parent-uuid>
    ├── related/
    │   └── <target-uuid>
    └── duplicate-of/
        └── <target-uuid>
```

Backward references and attribute indexes are NOT stored inside the track dir;
they live in the git-ignored `.gitrack/cache/` (§4.1).

- `<uuid>` is a UUIDv7 in canonical hyphenated form (ADR-0002). The track dir is
  named by it; the id is never a path fragment that changes.
- `slug` is an authored, human-chosen, repo-unique handle (kebab-case, derived
  from the title at creation, editable). Because tracks are keyed by UUID — not
  by a typeable git-ref — the slug is the handle humans type in references and
  URLs (ADR-0009). The UUID stays canonical; the slug is the ergonomic alias.
- Unknown files/dirs inside a track are preserved, not deleted — forward
  compatibility. A reader ignores what it does not understand.

There is deliberately **no** `referenced-by/` directory. It is derived (§8.1).

## 4. Relationships (ADR-0005)

Each relationship type is a directory; each edge is a file named by the target
UUID. The file may be empty or hold an optional note.

| dir (on A, pointing at B) | meaning                                        |
|---------------------------|------------------------------------------------|
| `depends-on/`             | A waits for B; A is blocked until B is closed  |
| `child-of/`               | A is contained by B (epic/project); B waits for A |
| `related/`                | soft link, no scheduling meaning               |
| `duplicate-of/`           | A is closed as a duplicate pointer to B        |

Inverse edges (`blocks`, `parent-of`) are **not stored**; tools compute them by
reversing the stored edges. One primitive (a directed link); the graph builder
branches only on link type to pick the edge **direction** in the blocking graph
(arrow points at the thing that must wait):

- `A depends-on B` → blocking edge `B → A` (A waits for B).
- `A child-of B` → blocking edge `A → B` (the parent B waits for the child A).

So `depends-on` and `child-of` block in **opposite** directions. Containment and
rollup read `child-of` edges only.

A track `A` is **unblocked** iff: every `B` with `A depends-on B` is closed, AND
every child `C` with `C child-of A` is closed. A leaf track with no `depends-on`
and no open children is unblocked. This is the input to boards/roadmaps.

Relationships are edited with the relation-named verbs — `git track depends-on
<ref> <target>`, `child-of`, `related`, `duplicate-of` — and removed with `git
track unlink <ref> <rel> <target>`. Each writes only the forward link; the
reverse `refs-in/` cache and blocked/unblocked status are derived. `child-of`
enforces the single-parent convention. `git track show` prints a track's
blocked/unblocked status computed from these edges.

`git track board` renders the unblocked-work view from these edges + state
(ADR-0027): open tracks grouped into **Ready** (no open blocker) and **Blocked**
(with the open blockers that hold them); closed tracks are omitted.
`board --mermaid` (default graph) and `board --dot` emit the blocking digraph
(edges point blocker → blocked, ready nodes marked) for rendering on GitHub, in
the web view, or with Graphviz.

## 4.1 The derived cache and `fsck` (ADR-0014)

Forward links above are the **source of truth**. Everything that inverts or
indexes them is a single **git-ignored local cache** under `.gitrack/cache/` —
rebuildable, never committed, never merged. It provides fast entrypoints:

- `cache/refs-in/<uuid>/<rel>/<referrer-uuid>` — who points at this track
  (reverse of the forward links; the basis for blocked/rollup queries).
- `cache/state/<state>/<uuid>`  — members currently in `<state>` (open/closed).
- `cache/status/<status>/<uuid>` — members currently in `<status>` (ADR-0019).
- `cache/type/<kind>/<uuid>`    — members of `<kind>`.
- future indexes live here too; the cache has no fixed schema.

Properties (all from ADR-0014):

- **Not committed.** `.gitrack/.gitignore` contains `/cache/`, so the cache never
  enters a commit and therefore never travels through a merge. The committed
  tree is purely authoritative (track objects + config).
- **Rebuildable.** The cache is a pure function of local HEAD's authoritative
  data. Each clone builds its own.
- **OS-adaptive.** Cache entries may be symlinks back to the track dir or plain
  files, chosen per-OS at runtime; readers must not assume which.
- **Never authoritative.** A track's own field always wins; a missing or stale
  cache only costs speed, never correctness.

Maintenance: every mutating command (`create|update|edit|close`) updates the
cache, and `gitrack fsck` rebuilds it entirely from the authoritative objects.
Because the cache is not in git, the committed-index merge-desync hazard that an
earlier design (ADR-0013) had to reconcile is **structurally impossible** — a
`post-merge` hook's only job is to rebuild the local cache, a safe local
operation. `fsck` also flags dangling forward links (to nonexistent tracks) and
unresolvable cross-repo refs (§8.1).

## 5. Comments

`comments/<uuid>.md` — one file per comment, append-only directory (concurrent
comments merge cleanly). `<uuid>` is a UUIDv7, so comments minted offline on
different branches never collide and the id is stable across edits (ADR-0023). A
content SHA was rejected: editing a comment would change its id and break
`reply-to` references.

A comment is **pure Markdown body**. It stores no author and no date — git
records both on the commit that added the file, and they are **derived**:

- author = `git log -1 --format='%an <%ae>' -- comments/<uuid>.md`
- created = `git log --diff-filter=A --format='%aI' -- comments/<uuid>.md`

The only metadata a comment may carry is **`reply-to`** (the parent comment's
UUID, for threading) — authored relationship data git cannot derive. A reply
carries a minimal front-matter block with only that key; a top-level comment has
no front matter at all:

```markdown
---
reply-to: 0192f1af-....           # only on a reply; parent comment UUID
---
The comment body in Markdown.
```

The thread tree is reconstructed from each comment's `reply-to`; absence means a
top-level comment. `get <ref>.comments` lists bare comment UUIDs (ADR-0022).

Comments are written with `git track comment <ref>`: `-m <msg>` for a
non-interactive body, or (no `-m`) `$EDITOR` opens on a temp file; `--reply-to
<cid>` sets the `reply-to` and seeds the editor with the parent cite-quoted as a
Markdown blockquote (git-commit style). The body is stored verbatim; any `#`refs
and URLs in it are rendered by the web/issues generator (ADR-0011), not at write
time.

## 6. Attachments

`attachments/<uuid>` holds the blob; `attachments/<uuid>.meta` is YAML front
matter describing it:

```yaml
name: screenshot.png             # original filename
type: image/png
# EITHER the blob is stored in-tree beside this .meta file (default),
# OR it points outside the tracker:
external: ../../design/mockup.png # optional: a path relative to repo root
```

External links let a track reference a file already in the repo (or elsewhere)
without duplicating it into the tracker.

## 7. Configuration (`.gitrack/config/`)

Config follows the same key/value rule — it is itself tracked data, so workflow
rules merge with the code.

```
.gitrack/config/
├── kinds                 # allowed kinds, one per line (cosmetic, ADR-0006)
└── statuses/             # one file per user-defined status (ADR-0019)
    └── <status>          #   that status's allowed next statuses, one per line
```

### State and status: two axes (ADR-0019)

A track has two single-valued lifecycle fields:

- **`state`** — fixed, never customizable: `open` or `closed`. A track is closed
  iff `state == closed`.
- **`status`** — the customizable workflow position. Three statuses are **built
  in and always valid**: `new`, `reopened`, `done`. Every other status is
  user-defined as a file `config/statuses/<name>` whose lines are that status's
  **allowed next statuses**; the built-ins may also declare theirs. A status with
  no file, or an empty one, permits any transition — enforcement is opt-in.

The two axes are kept coherent by the lifecycle verbs:

- `git track create` -> `state=open`, `status=new`.
- `git track update -s <status>` -> moves `status` through the workflow (state
  unchanged); a disallowed next status is rejected.
- `git track close [--status=<s>]` -> `state=closed`, `status=<s>` or `done`.
- `git track reopen` -> `state=open`, `status=reopened`.

Example `config/statuses/in-progress`:

```
in-review
done
```

Workflows are edited with `git track config status` (ADR-0025), not by hand:
`config status add <name> [--next a,b,c]`, `config status set-next <name>
[next...]`, `config status list`, `config status rm <name>`. Transition targets
are validated to be existing statuses; the three built-ins cannot be removed.

## 8. Task lists in descriptions (ADR-0008)

A `description` may contain GitHub-Flavored-Markdown task lists. Two item kinds:

- **Free item** — `- [ ] write the release notes`. Hand-authored; the checked
  state is prose the user toggles by editing `description`. gitrack never
  touches it.
- **Dependency-backed item** — `- [ ] #<ref> short label`, where `#<ref>` is a
  reference (§8.1: full/short UUID, slug, or cross-repo). Its
  rendered checked state is **derived**: `[x]` iff the referenced track is
  closed, else `[ ]`. The authored bytes stay `- [ ]`; the rendered page shows
  the computed box. Creating/editing such an item also materializes a matching
  `depends-on/<uuid>` link, so the checklist and the graph are one source of
  truth. Closing the referenced track changes only its own `state` file, yet
  every dependent checklist shows it checked on next render — no cascading
  writes (ADR-0007/0009).

A plain GitHub preview still shows a normal task list; the derived checking is a
gitrack-render enhancement, so the format degrades gracefully.

## 8.1 Reference syntax (ADR-0011)

A reference starts with `#` and resolves to a track UUID. Tracks are keyed by
UUID (not a typeable git-ref), so several typeable forms resolve to one:

| form                        | resolves to   | notes |
|-----------------------------|---------------|-------|
| `#<uuid>`                    | that UUID     | full canonical id |
| `#<short-uuid>`              | a UUID        | shortest unique hex prefix, git-short-hash style; ambiguity errors with candidates |
| `#<slug>`                    | a UUID        | authored `slug` or an alias |
| `#<repo-url>?track=<uuid>`   | a remote UUID | cross-repo (the `<uuid>` may be short) |

Slug vs short-uuid is disambiguated by shape (a pure-hex value is a short-uuid; a
slug may not be pure hex — enforced at mint). From a code commit, the
`Gitrack-Ref: <ref>` trailer carries the same value with the `#` dropped.

Cross-repo refs resolve against the named remote when reachable; offline, the
renderer shows the URL plus short id, and `fsck` (§4.1) refreshes them.

## 8.2 Accessor paths — `get` / `set` (ADR-0021)

Because a field is a file and a multi-value attribute is a directory, a track is
addressable by a dotted path `<ref>.<key>[.<key>...]` that maps straight onto
the tree. The first `.` separates the object reference (uuid/short/slug — none
contain `.`) from the key path; further `.` segments descend.

- `get [--json] <ref>` — the whole track (object).
- `get <ref>.<field>` — a single-valued field (scalar), e.g. `<ref>.status`.
- `get <ref>.<attr>` — a multi-value attribute lists its entries as **bare ids**,
  e.g. `<ref>.comments`, `<ref>.labels`, `<ref>.depends-on`. Any storage suffix
  (see below) is stripped from the listing.
- `get <ref>.<attr>.<id>` — a nested entry by bare id. If the attribute declares
  a storage suffix, it is appended to find the file; resolution never strips an
  arbitrary extension (ADR-0022). So `comments.<uuid>` reads
  `comments/<uuid>.md` because `.md` is the comments attribute's declared suffix.

Storage suffix: a multi-value attribute may store its entries with a fixed
suffix marking the on-disk file type — `comments` entries are `<id>.md`
(Markdown). The suffix is a storage detail, never part of the id, the command,
or the listing (ADR-0022).
- `set <ref>.<key> <value>` — writes a **free-form** field only (`title`,
  `description`, `kind`, ...). The **governed** fields `state` and `status` are
  refused with a redirect to `close`/`reopen` and `update --status`, so the
  lifecycle (ADR-0019) keeps a single validated path. `update` remains the
  human-friendly, invariant-aware multi-field editor.

In rendered output (the `web` generator), a `#<ref>` in a description — a slug,
short-uuid, or full uuid — is **autolinked** to that track's page when it
resolves to an existing track (ADR-0030); a non-resolving `#` stays plain text.
Links are relative, so they work offline and from a moved site. Comment bodies
use the same autolinker when rendered.

`git track commit [<id>...] [-m <msg>] [--status <s>] [-- <file>...]` processes
the **commit message's own directives** and bundles the result as one revision
(ADR-0033). gitrack obtains the message (from `-m`, else `$EDITOR`), parses every
`<verb> #<handle>` in it — `closes`/`resolves`/`fixes` → **close the track**,
`relates to`/`related to` → relate (informational), bare `#<id>` → mention —
applies the closing directives, stages the tracker changes, and commits. With
`-- <file>...` the commit is scoped to those files plus `.gitrack/`; with no
pathspec it commits the staged index. A positional `<id>` (or `--resolve <id>`)
is sugar that injects a `closes #<slug>` directive; `--status` sets the closing
status. So writing `closes #login` in the message and committing through gitrack
closes the track in the same commit as the code. Passively observing a `closes`
commit never auto-closes (ADR-0031); state changes only via `git track close` or
this command.

A track's **mentions** are derived from git history (ADR-0031, never stored):
commits whose message references the track via `#<handle>` — `closes #<id>` /
`resolves` / `fixes` (verb `closes`), `relates to #<id>` / `related to` (verb
`relates`), or a bare `#<id>` (verb `mentions`). `git track show` and the web
track page list them (short-sha + verb + subject), so the source tree
cross-references the tracker. A `closes #<id>` commit does **not** auto-close the
track — gitrack has no server; closing is the explicit `git track close`.

## 9. Derived views (the "dumb code" contract)

Authored data is read directly; derived data is computed, never stored
(ADR-0009). A reader:

- **Field value:** read the file. Done.
- **Multi-value attribute:** list the directory. Done.
- **Closed?** read `state`; closed iff `state == closed` (ADR-0019).
- **Unblocked?** every target in `depends-on/` is closed, AND every child
  (track whose `child-of/` points here) is closed (ADR-0005 directions).
- **Comment thread:** read each `comments/*.md` front matter, order by `reply-to`.
- **`referenced-by`** (DERIVED, never stored): the union of
  - `git log -- .gitrack/<uuid>/` — commits that touched this track (structural;
    needs no typed reference, cannot drift), and
  - `git log --grep='Gitrack-Ref: <ref>'` for the track's slug and aliases —
    code commits that explicitly opted in (ADR-0011 trailer).
- **Inverse edges** (`blocks`, `parent-of`), **rollup/progress**, and
  **dependency-backed checkbox state:** all computed from authored data.

No fold, no replay, no authoritative cache. The git-ignored `.gitrack/cache/`
(§4.1) memoizes these derived results for speed and is always regenerable by
`gitrack fsck`; it is never the source of truth.

## 9.1 Live preview — `git track serve` (ADR-0024)

`git track serve` starts a **loopback-only** HTTP server on a random port
(default `127.0.0.1:0`) and prints its URL, in the spirit of `hugo server`. It
renders the **current tracker state live** — each request reads the authoritative
tree, so a track created or changed after startup appears on the next request
with no restart. Routes: `/` (track index), `/track/<id>` (one track), and
`/board` (the Ready/Blocked view plus the dependency digraph rendered in-browser
via Mermaid, ADR-0027). It is read-only (no mutation endpoint) and refuses any
non-loopback bind. The Git-Web source-code view and live-reload push are planned
follow-ons.

## 9.2 Static site — `git track web` (ADR-0029)

`git track web --out <dir>` generates a **self-contained static site** from the
tracker: an index of tracks, a page per track, and a board page. Per ADR-0029
the output is offline-capable and host-neutral — inline CSS (no external
stylesheet), all links relative (works from `file://` or any subpath), and no
network dependency for core content. The board's dependency digraph is a
progressive enhancement (Mermaid) sitting above the always-present HTML
Ready/Blocked lists, so the page stays usable offline.

Because the output is a plain directory with no host coupling, it deploys the
same way everywhere: `aws s3 sync`, a copy to a shared drive, `rsync`, an IPFS
add, or the GitHub Pages workflow (ADR-0010) — the decentralized, no-hosted-
service story (docs/VISION.md). `git track serve` is the live local preview;
`git track web` is the deployable snapshot.

## 10. Publishing (per-commit static site)

The tracker is meant to replace a hosted issue tracker, so it ships a CI
workflow that rebuilds and deploys a static site on every commit touching the
tracker (`on: push`, `paths: ['.gitrack/**']`), using the standard GitHub Pages
deploy actions. The generated site reads only this format and is purely derived;
see `docs/adr/0010-publishing.md` and the web/issues generators (later work).

## 11. Open questions to resolve through scenarios

- **Resolved (ADR-0019):** two axes — a fixed `state` (`open`/`closed`) changed
  only by `new`/`close`/`reopen`, and a customizable `status` workflow (built-ins
  `new`/`reopened`/`done` plus user-defined statuses with per-status transitions)
  moved by `update -s`.
- Should `created`/author of the track itself live in files, or is the first
  commit touching the track dir authoritative? (git already records who/when.)
- History of a single-value field: currently only the latest value is kept
  (git log recovers history). Is that sufficient, or does any field need an
  in-tree audit trail like comments?
- Interactive/TUI mode for `create`/`update` (gh-style prompts) — specified as a
  goal; not yet designed or built (the non-interactive verbs come first).

The remaining items are left open; we will settle them by working scenarios.
