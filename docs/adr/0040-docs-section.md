# ADR-0040: Docs section with a configurable navigation layout

## Status
accepted

## Context
ADR-0037 lists `docs` as a section of the full project website: "the repo's
Markdown documentation with a project-defined layout." ADR-0038 supplies the
shared offline Markdown renderer with portal-wide `#ref` autolinking, and
ADR-0039 established the content/section patterns (authored Markdown, front
matter, title fallbacks). This ADR specifies the `docs` section.

Docs differ from portal/blog in two ways:
1. **Source location.** Documentation is the project's *existing* `docs/`
   Markdown (this very repo keeps ADRs and VISION there), not new content
   authored under `.gitrack/config/site/`. The section renders what the project
   already writes.
2. **Navigation is a project-defined tree.** Unlike the flat portal nav, docs
   need an ordered, possibly nested sidebar the project controls — the
   "configurable nav layout" in ADR-0037.

## Decision

### Source: a configured docs directory (default `docs/`)
The docs section renders Markdown from a directory **relative to the repo root**,
set by `.gitrack/config/site/docs/source` (a single path line). Absent ⇒ default
`docs`. Only `*.md` files are rendered; other files are ignored. A missing source
directory ⇒ the section emits nothing and drops its nav entry (ADR-0026).

Each doc file at `<source>/<rel>.md` renders to `docs/<rel>.html` in the output
(the `docs/` output prefix is fixed regardless of source dir name, so links are
stable). Bodies render through `renderMarkdown` (ADR-0038), so `#refs` and
relative Markdown links work; a landing page is written to `docs/index.html`
(see Layout).

### Published docs are not all repo docs: internal docs are excluded
The repo's `docs/` tree mixes **user-facing documentation** with **internal
design records** — ADRs, prior-art surveys, on-disk-format specs — that are
written for contributors, not for users of the tooling. The published docs site
must render the former and omit the latter.

Exclusions are declared in `.gitrack/config/site/docs/exclude`, one path pattern
per line, matched against each doc's rel path (slash-separated, no extension). A
pattern matches a doc when it equals the rel path, equals a leading path segment
of it, or is a `*` glob on the base name:
- `adr` or `adr/` excludes the whole `adr/` subtree;
- `CONTRIBUTING` excludes that one doc;
- `*-internal` excludes docs whose base name ends in `-internal`.

**Default when the file is absent:** exclude `adr` (the ADR subtree), because
ADRs are internal design records by strong convention and should not appear on a
project's user-facing docs site. A project that *does* want to publish its ADRs
sets an explicit `exclude` file (empty, or without `adr`); a project with other
internal trees adds them. Excluded docs stay in the repo and in git history —
they are simply not rendered into the site and never appear in the sidebar
(including the auto "Other" group) or autolink doc targets.

### Navigation layout: project-defined, with an auto fallback
The sidebar is declared in `.gitrack/config/site/docs/layout`, an ordered,
indentation-nested list of entries. Each non-blank, non-`#` line is:

```
[indent]<path-or-label>[: <label>]
```

- A line whose token ends in `.md` (or names an existing doc) is a **page**
  entry: it links to that doc, labeled by the optional `: <label>`, else the
  doc's title (front matter / H1 / file base, ADR-0039).
- A line that does not resolve to a doc is a **group heading** (a non-link
  label); its indented children nest under it.
- Indentation (leading spaces/tabs, two spaces or one tab per level) sets
  nesting depth, so the layout is a tree.

The layout is a guardrail, not enforcement (ADR-0026): a layout entry pointing
at a missing doc is rendered as a plain (dead-safe) label and reported by fsck
later, never a generation error; a doc on disk not mentioned in the layout is
still rendered to its page and appended to the sidebar under an "Other" group so
nothing is silently unreachable.

**Absent layout ⇒ auto layout:** generate the sidebar from the source tree —
subdirectories become groups, files become entries, ordered alphabetically
(an `index.md`/`README.md` in a directory, if present, is that group's landing
link and sorts first). This gives a usable docs site with zero config, while a
project that wants control writes a `layout` file.

### The docs landing page
`docs/index.html` is: the source dir's top-level `index.md`/`README.md` rendered,
if present; otherwise an auto landing that just shows the sidebar. The docs nav
entry in the top bar points at `docs/index.html`.

### Sidebar rendering (no theme change)
The sidebar (the layout tree) is rendered **into each docs page body** as a
`<nav class="docs-nav">`, so the docs section needs no change to the theme shell
(which carries only the top section bar). Every docs page shows the top bar
(sections) plus the docs sidebar. The default theme gains inline CSS for the
sidebar (ADR-0029 offline), additive like ADR-0039's Markdown CSS.

### Top-bar nav
The docs section contributes one top-bar entry: `docs` → `docs/index.html`.

## Consequences
- Easier: a project's existing `docs/` renders into the themed, autolinked,
  offline site with a sidebar it controls, or a sensible auto one with no config.
- Easier: the `docs/` output prefix is fixed, so inbound links and the top-bar
  entry are stable regardless of the configured source directory name.
- Harder: a layout parser (indentation tree + page/group classification) and an
  auto-layout walker; both are small and data-driven (smart data, dumb code).
- Safe: unmatched layout entries degrade to labels (never break generation), and
  unlisted (but not excluded) docs are still rendered and surfaced under "Other"
  so nothing user-facing is lost.
- Right default: ADRs and similar internal records are excluded from the
  published site out of the box, so a project does not accidentally ship its
  design history to its users; publishing them is an explicit opt-in.
- Cost accepted: nested relative links between docs pages are the doc author's
  responsibility (standard Markdown); the section does not rewrite intra-doc
  links beyond resolving them relative to each page's output location.

## Verification
- [ ] With `docs` enabled and a `docs/` source, every `*.md` renders to
      `docs/<rel>.html`; `docs/index.html` exists (from index/README or auto).
- [ ] `.gitrack/config/site/docs/source` redirects the source dir; the output
      prefix stays `docs/`.
- [ ] A `layout` file sets sidebar order/nesting; group headings nest their
      children; a page entry links to its doc with the chosen/derived label.
- [ ] A layout entry pointing at a missing doc renders as a plain label (no
      error); a doc not in the layout (and not excluded) still renders and
      appears under "Other".
- [ ] With no `exclude` file, the `adr` subtree is omitted by default: no
      `docs/adr/*.html` pages, no ADR entries in the sidebar. An explicit
      `exclude` file overrides the default (empty ⇒ publish everything).
- [ ] Absent `layout` ⇒ an auto sidebar from the directory tree (dirs as groups,
      alphabetical, index/README first).
- [ ] `#refs` in doc bodies autolink (ADR-0038); the sidebar and top bar both
      render with correct relative hrefs from nested doc pages; output is offline.
- [ ] Absent source dir ⇒ the section emits nothing and leaves no top-bar entry.
