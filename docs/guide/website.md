# The project website

`git track web` renders your whole project as one self-contained static site —
issues, marketing pages, a blog, and documentation — that deploys to any dumb
host or opens straight from `file://`. `git track serve` is the live local
preview.

## Generate a site

```sh
git track web --out ./site
```

The output is a plain directory with no host coupling: inline CSS (no external
stylesheet), relative links (works from any subpath or `file://`), and no
network dependency for core content. Deploy it by copying the directory anywhere
(see [Publishing](publishing.md)).

## Live preview

```sh
git track serve                 # random loopback port
git track serve --addr 127.0.0.1:8080
```

`serve` binds a loopback-only port, prints its URL, and renders the **current**
tracker on every request — a track you create or change afterward appears on the
next request with no restart. It is read-only and refuses any non-loopback bind.

## Sections

The site is built from independent **sections** over a shared theme. You choose
which are enabled in `.gitrack/config/site/sections` (one name per line). With
no config, only the `issues` section is on, so the site is exactly the tracker
view. The available sections:

| section | renders |
|---|---|
| `issues` | the tracker: tracks index, issues (open/closed), board, per-track pages with threaded comments |
| `portal` | a front page, plus about/contact and any other marketing pages |
| `blog` | a news blog: one page per post and a newest-first index |
| `docs` | the project's Markdown documentation with a configurable sidebar |

(Source-tree browsing and API reference sections are planned.)

Example `.gitrack/config/site/sections`:

```
portal
blog
docs
issues
```

A section with no content emits nothing and drops its nav entry, so enabling a
section you haven't populated yet is harmless.

## The home page

The site's `index.html` belongs to the **first enabled section that can produce
a home page**. With issues only (the default), the tracks index is the home
page. Enable `portal` with a `front.md` and the portal front page becomes home,
while the tracks index moves to `tracks.html`. A portal without a `front.md`
declines the home slot, so home falls through to the next section.

## Portal and blog content

Both live under the site config directory as authored Markdown:

- Portal pages: `.gitrack/config/site/portal/*.md`. `front.md` is the front
  page; every other file becomes `portal/<name>.html` with a nav entry.
  `about.md` and `contact.md` are just portal pages.
- Blog posts: `.gitrack/config/site/blog/*.md`, rendered to `blog/<slug>.html`
  with a `blog/index.html` listing them newest-first.

A page or post may start with a tiny front-matter block:

```markdown
---
title: Release 1.0 is out
date: 2026-11-04
---

Body in **Markdown**. A #ref here autolinks to its track.
```

`title` sets the page/post title (otherwise the first `# H1`, then the file
name); `date` (blog only) orders the index newest-first. Front matter is
optional — a bare Markdown file works.

## Docs section

The `docs` section renders your project's documentation and is covered in depth
by the docs it produces, but in short:

- **Source**: a repo-relative directory, default `docs/`, set by
  `.gitrack/config/site/docs/source`. Output always lands under `docs/`.
- **Sidebar**: a project-defined tree in `.gitrack/config/site/docs/layout`
  (indentation-nested page/group entries), or an auto sidebar from the directory
  tree when absent.
- **Exclusions**: `.gitrack/config/site/docs/exclude` lists path patterns to
  omit from the published site. By default the `adr/` subtree is excluded, since
  architecture decision records are internal, not user-facing. Set an explicit
  (even empty) `exclude` file to publish everything.

See [Configuration](configuration.md) for the exact config files.

## What's static and what's enhanced

Everything the site needs to be usable is static HTML with inline CSS. The only
enhancement is the board's dependency graph, drawn by Mermaid when the page has
network access, sitting above the always-present Ready/Blocked lists. Offline or
on a dumb host, the site is fully functional; the graph is a bonus.
