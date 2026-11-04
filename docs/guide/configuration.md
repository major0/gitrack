# Configuration

All configuration lives under `.gitrack/config/` as plain files that follow the
same key/value rule as the rest of the tracker — so your workflow rules and site
settings are versioned and merge with the code. Config is a **guardrail, not
enforcement** (see [Status workflows](workflow.md)): it shapes gitrack's
well-behaved path but cannot constrain what the tree physically contains, and
`git track fsck` reconciles divergence with advisory warnings.

## Layout

```
.gitrack/config/
├── kinds                 # allowed track kinds, one per line (cosmetic)
├── statuses/             # one file per user-defined status
│   └── <status>          #   that status's allowed next statuses, one per line
└── site/                 # static-site generator settings
    ├── sections          # enabled site sections, one per line
    ├── theme             # theme name (built-in default when absent)
    ├── portal/           # portal pages (*.md); front.md is the front page
    ├── blog/             # blog posts (*.md)
    └── docs/             # docs section settings
        ├── source        # repo-relative docs source dir (default "docs")
        ├── layout        # docs sidebar tree
        └── exclude       # doc path patterns to omit from the site
```

## Kinds

`.gitrack/config/kinds` lists the allowed `kind` values, one per line (e.g.
`project`, `epic`, `issue`, `bug`, `task`, `note`). It is cosmetic and advisory:
an absent file means any kind is allowed.

## Statuses

`.gitrack/config/statuses/<name>` defines a user-defined status; its lines are
that status's allowed next statuses. Edit these with `git track config status`
rather than by hand — see [Status workflows](workflow.md). The built-in
statuses `new`, `reopened`, and `done` are always valid and cannot be removed.

## Site sections

`.gitrack/config/site/sections` lists the enabled sections in order, one per
line (`portal`, `blog`, `docs`, `issues`). Blank lines and `#` comments are
ignored. Absent ⇒ `issues` only, preserving the plain tracker view. An
unknown or not-yet-built section name is skipped.

## Theme

`.gitrack/config/site/theme` names the theme; absent (or `default`) uses the
built-in theme, whose CSS is inlined so the site works offline.

## Portal and blog content

- `.gitrack/config/site/portal/*.md` — one marketing page per file. `front.md`
  is special (the front page). Other files become `portal/<name>.html`.
- `.gitrack/config/site/blog/*.md` — one post per file, listed newest-first by
  the `date` front-matter key.

Both support an optional `title:`/`date:` front-matter block. See
[The project website](website.md).

## Docs section

Three files under `.gitrack/config/site/docs/`:

### `source`

A single path line, relative to the repo root, naming the documentation
directory. Absent ⇒ `docs`. The rendered output always lands under the fixed
`docs/` prefix regardless of the source dir name, so inbound links stay stable.

### `layout`

The sidebar tree, an indentation-nested list (two spaces or one tab per level):

```
index.md: Home
Guide
  guide/getting-started.md: Getting started
  guide/concepts.md: Concepts
Reference
  reference/commands.md: Commands
```

- A line naming a known doc (with or without `.md`) is a **page** entry; the
  optional `: label` overrides the derived title.
- A line that doesn't resolve to a doc is a **group heading** (plain text); its
  indented children nest under it.
- A layout entry pointing at a missing doc degrades to a plain label — never an
  error.
- A doc on disk not listed in the layout is still rendered and appended under an
  auto **Other** group, so nothing user-facing is unreachable.

Absent ⇒ an auto sidebar generated from the directory tree (subdirectories as
groups, alphabetical, `index`/`README` first).

### `exclude`

Path patterns (one per line) for docs to omit from the published site — they
stay in the repo and git history, just not on the site, the sidebar, or as
autolink targets. A pattern matches an exact rel path, a leading path segment
(`adr` ⇒ the whole `adr/` subtree), or a base-name glob (`*-internal`).

**Default when the file is absent: the `adr/` subtree is excluded**, because
architecture decision records are internal design docs, not user-facing. To
publish everything, add an explicit `exclude` file (empty is fine); to hide more
internal trees, list their patterns.
