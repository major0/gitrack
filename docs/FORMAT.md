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
│   ├── state/<state>/<uuid>     #   index: tracks by state
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

## 4.1 The derived cache and `fsck` (ADR-0014)

Forward links above are the **source of truth**. Everything that inverts or
indexes them is a single **git-ignored local cache** under `.gitrack/cache/` —
rebuildable, never committed, never merged. It provides fast entrypoints:

- `cache/refs-in/<uuid>/<rel>/<referrer-uuid>` — who points at this track
  (reverse of the forward links; the basis for blocked/rollup queries).
- `cache/state/<state>/<uuid>` — members currently in `<state>`.
- `cache/type/<kind>/<uuid>`   — members of `<kind>`.
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
comments merge cleanly). YAML front matter carries threading and authorship:

```markdown
---
id: 0192f1b0-....                 # this comment's UUID
author: Mark Ferrell <mark@example.com>
created: 2026-11-02T16:45:00Z
reply-to: 0192f1af-....           # optional: parent comment UUID for threading
---
The comment body in Markdown.
```

`reply-to` gives threaded ordering without a mutable index: the thread tree is
reconstructed by reading each comment's front matter. Absence of `reply-to`
means a top-level comment.

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
├── kinds                 # allowed kinds, one per line (project, epic, issue, bug, task, note)
├── states                # allowed states, one per line
├── states.closed         # subset of states that count as "closed", one per line
└── workflow/
    └── <kind>            # optional: allowed state transitions for this kind,
                          #   one "from -> to" per line; absent = unrestricted
```

Per-kind workflow example, `.gitrack/config/workflow/bug`:

```
new -> triaged
new -> wontfix
triaged -> in-progress
in-progress -> in-review
in-review -> done
```

A kind with no workflow file has unrestricted transitions. "Supported tracks,"
"states per track type," and "workflows per track type" are all expressed here
as plain files.

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

## 9. Derived views (the "dumb code" contract)

Authored data is read directly; derived data is computed, never stored
(ADR-0009). A reader:

- **Field value:** read the file. Done.
- **Multi-value attribute:** list the directory. Done.
- **Closed?** read `state`, test membership in `config/states.closed`.
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

## 10. Publishing (per-commit static site)

The tracker is meant to replace a hosted issue tracker, so it ships a CI
workflow that rebuilds and deploys a static site on every commit touching the
tracker (`on: push`, `paths: ['.gitrack/**']`), using the standard GitHub Pages
deploy actions. The generated site reads only this format and is purely derived;
see `docs/adr/0010-publishing.md` and the web/issues generators (later work).

## 11. Open questions to resolve through scenarios

- Should `state` and `status` be one field or two (coarse open/closed vs
  fine-grained workflow step)? gitrack v1 had both.
- Should `created`/author of the track itself live in files, or is the first
  commit touching the track dir authoritative? (git already records who/when.)
- History of a single-value field: currently only the latest value is kept
  (git log recovers history). Is that sufficient, or does any field need an
  in-tree audit trail like comments?

These are deliberately left open; we will settle them by working scenarios.
