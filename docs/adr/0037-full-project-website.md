# ADR-0037: The site generator is a full project website, not just a dev portal

## Status
accepted (supersedes the scope boundary of ADR-0036)

## Context
ADR-0036 defined `git track web` as a themeable, config-driven **developer
portal** — issues + docs + source — and drew an explicit boundary: it is NOT the
project's marketing/primary website, for which separate tooling such as Hugo was
expected.

The stakeholder has now removed that boundary. gitrack's site generator should
produce the **whole project website**, so a project needs no second static-site
tool. The requested surface is:

- **Web portal / marketing pages** — front page, about, contact, and a **news
  blog**.
- **Docs** — the project's Markdown documentation, with a **configurable layout**
  (section tree / navigation the project defines).
- **API reference** — generated from source, with **language-specific
  generators** (e.g. Go doc, and other languages as plug-in generators).
- **Issues** — the tracker (EXISTS: ADR-0027/0030/etc.).
- **Releases** — release notes tied to tags, autolinking the issues they ship.
- **Source browsing** — a gitweb-style commit/tree/blob browser.

Autolinking of `#refs` (ADR-0011/0030/0034) must work across *every* section
that renders authored text (docs, blog, releases, issues, API prose, source
commit messages), not just issue descriptions and comments.

This keeps ADR-0036's mechanism — independent section renderers over a themeable
shell, config-driven, static/offline/host-neutral (ADR-0029) — and only widens
the set of sections and the goal. The "not the marketing site" sentence in
ADR-0036 is superseded; everything else in ADR-0036 stands.

## Decision
`git track web` generates a complete, static, offline, host-neutral project
website built from independent **sections** over a shared **theme**, selected and
configured by files under `.gitrack/config/site/` (guardrails-not-enforcement,
ADR-0026). The section catalog:

| Section    | Produces                                                        | Source of truth                         |
|------------|-----------------------------------------------------------------|-----------------------------------------|
| `portal`   | front page, about, contact (static marketing pages)             | Markdown under a configured content dir |
| `blog`     | reverse-chronological news posts + index/feed                   | dated Markdown posts                    |
| `docs`     | documentation tree with a project-defined nav layout            | repo Markdown + a layout config         |
| `api`      | API reference per language, via pluggable generators            | source code (language-specific)         |
| `issues`   | tracker: index, issues, board, per-track (EXISTS)               | `.gitrack/` tracks                       |
| `releases` | release notes per tag, autolinking shipped issues               | `release` objects (new kind)            |
| `source`   | gitweb-style commit / tree / blob browser                       | the git repository                      |

Cross-cutting rules:

1. **One section framework.** A section is a renderer with a stable contract
   (name, nav label, enabled?, and `Render(ctx) → files`). `Generate` builds the
   enabled set, renders each into the theme shell, and assembles the nav. The
   current issues/board/track output becomes the `issues` section renderer; the
   hardcoded `baseTmpl` becomes the default theme shell. This refactor lands
   first and must be behavior-preserving (no site config ⇒ today's output).

2. **Autolinking is portal-wide and shared.** `web.autolink` (escape-then-linkify
   resolvable `#refs`, ADR-0030/0034) is the single pass every section routes its
   authored text through. Markdown sections run it as a post-render step over
   rendered prose nodes only (never headings-as-anchors or code). Resolution is
   extended so a `#ref` may target a track, a release, or a doc/page — all as
   relative links (ADR-0029). This is specified in a follow-on ADR (Markdown +
   autolink composition) because it forces the Markdown-rendering decision
   ADR-0030 deferred.

3. **Markdown rendering is offline and pure-Go.** Every Markdown section (portal,
   blog, docs, release notes) shares one renderer: pure-Go, no CDN, CSS inlined
   by the theme (ADR-0029). The library choice is pinned in the Markdown ADR.

4. **API reference is a pluggable, language-keyed generator set.** `api` dispatches
   per language to a generator (Go first, via `go/doc`); other languages register
   as generators keyed by config. A project with no supported language simply
   emits no `api` section. Generators run offline against local source only.

5. **`releases` needs a `release` object kind.** A release is a tracked object
   (tag name + date-from-git + Markdown notes + `closes` refs), reusing the
   storage (ADR-0003) and commit (ADR-0028) models, specified in its own ADR.

6. **Config under `.gitrack/config/site/`** selects sections (`sections`), theme
   (`theme`), content locations (e.g. `portal/`, `blog/`, docs layout), and
   per-language API/highlight options (`lang/<ext>`). Absent config = issues-only,
   default theme — existing behavior unchanged.

7. **Static / offline / host-neutral (ADR-0029)** is preserved for the whole
   site: a self-contained directory deployable by file copy / `aws s3 sync` /
   Pages and openable from `file://`. Progressive enhancements (e.g. Mermaid,
   search) stay above an always-present static fallback.

This ADR is the umbrella architecture. Each section and the shared Markdown/
autolink/theme machinery land incrementally behind their own ADRs, each step
green and behavior-preserving by default.

## Consequences
- Easier: a project ships its entire web presence — marketing, blog, docs, API,
  issues, releases, source — from one offline generator; no second static-site
  tool required.
- Easier: one autolink pass and one Markdown renderer serve every section, so a
  `#ref` resolves the same way everywhere.
- Harder: this is a substantially larger subsystem than ADR-0036 scoped. It is
  built section by section; the enabling refactor (section framework + theme +
  Markdown + portal-wide autolink) comes before any new section.
- Harder: the `api` section couples to language toolchains; it is pluggable and
  optional so unsupported languages degrade to "no api section", and generators
  stay offline against local source.
- Cost accepted: gitrack now competes with general static-site generators for the
  project-website use case. We accept that scope deliberately; the differentiator
  remains that the site is generated from the repo + tracker with zero-infra,
  offline, host-neutral output and repo-native autolinking.

## Verification
- [ ] `docs/VISION.md` states the full-website goal (portal/blog/docs/api/issues/
      releases/source), superseding the "not the marketing site" boundary.
- [ ] A section framework exists; the issues/board/track output is produced by an
      `issues` section and, with no site config, the generated site is byte-for-
      byte today's output (issues on, default theme).
- [ ] `.gitrack/config/site/sections` enables/disables sections; a disabled
      section emits nothing and leaves no nav entry.
- [ ] Autolinking resolves `#refs` to tracks, releases, and doc pages across every
      section that renders authored text.
- [ ] Each new section (portal, blog, docs, api, releases, source) plugs into the
      framework behind its own ADR, each landing green.
