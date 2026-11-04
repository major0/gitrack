# Vision

gitrack exists to let a team run a full issue/task/note tracker **without a
hosted service** — the way it used to be, before every workflow required a
server someone else owns.

## The north star

Your tracker is plain files in your git repository. Git already goes everywhere:
a laptop, a USB stick, a shared drive, an S3 bucket, a Dropbox folder, an IPFS
pin, an email attachment (`git bundle`). gitrack adds nothing that needs a
server of record. When you clone the repo, you have the whole tracker —
history, issues, comments, relationships, boards — offline and complete.

## Why decentralized + static

- **No hosted dependency.** There is no gitrack server to run, pay for, patch,
  or trust. The data lives with the code and travels by whatever moves git:
  https, ssh, a filesystem remote on a shared drive, a `git bundle` on a
  drop-box, an S3-backed remote.
- **Static output, dumb hosts.** The web and issues views, boards, and roadmaps
  render to static HTML and assets. Static files serve from anything — an S3
  bucket, a shared drive, GitHub/GitLab Pages, IPFS, or just `file://` in a
  browser. No application server means a far smaller attack surface: there is no
  running endpoint to exploit, no database to inject, no auth to bypass.
- **Less energy.** A static site answered from a bucket or a CDN edge, or opened
  straight off disk, costs a fraction of the energy of an always-on hosted
  application. Decentralizing to static content is a small but real
  sustainability win — save the planet, one un-run server at a time.
- **Resilient and private.** Offline-first by construction; no third party sees
  your issues; no single host to take the whole team's tracker down.

## What this commits the design to

- The on-disk format is the source of truth and is portable by git alone;
  nothing requires a central service.
- Every rendered view must be **static** and deployable to a dumb host or opened
  as `file://` — no runtime server in the deployment path. `git track serve` is
  a local *preview* convenience, never the deployment.
- Publishing targets are transport-agnostic: GitHub Pages today, and
  S3 / shared drive / IPFS / bundle as first-class static-sync targets over time.

## The project website (the site generator's shape)

`git track web` renders the **whole project website** as one themeable static
site, so a project needs no second static-site tool. It is built from
independent, config-toggled sections over a shared theme:

- **Issues** — the tracker: index, issues, board, and per-track pages with
  threaded comments. Built.
- **Portal** — a front page, about, contact, and other marketing pages. Built.
- **Blog** — a news blog, newest first. Built.
- **Docs** — the repo's Markdown documentation with a project-defined sidebar
  layout. Built.
- **Releases** — release notes per tag, with the tracks each release shipped.
  Built.
- **Source tree** — a gitweb-style browser of commits, trees, and files.
  Planned.
- **API reference** — generated from source via language-specific generators
  (Go first). Planned.

Simple config under `.gitrack/config/site/` enables or disables each section and
says where its content lives. The shared page shell is one place to restyle the
whole site, and the built-in theme ships inline so the site works offline from
`file://`. Swapping in a custom theme and per-language options (highlighting,
API layout) are planned. Autolinking of `#refs` works across every section that
renders authored text — issues, portal, blog, docs, and releases — not just
issues.

The goal is the project's **complete web presence** — marketing, blog, docs,
issues, releases, source, and API — generated from the repo and tracker with
zero-infra, offline, host-neutral output. Source browsing and the API reference
are what remain.

## What it does NOT mean

- Not anti-collaboration: you still sync with teammates — through git remotes,
  which can themselves be decentralized (a shared drive, an S3 remote), not a
  hosted tracker.
- Not anti-GitHub: GitHub Pages is a fine static host, and the `#<repo-url>`
  cross-repo reference spans repositories wherever they live.
- Not an application framework: the site is fully static and offline by
  construction. It renders the project's whole web presence from the
  repo and tracker, but it runs no server in the deployment path.
