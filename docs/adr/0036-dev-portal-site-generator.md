# ADR-0036: The web output is a themeable, config-driven developer portal

## Status
accepted

## Context
The original goal #3 was "a statically rendered website for the git side of the
repository — gitweb, but with the flexibility of Hugo." The stakeholder has now
sharpened that vision: gitrack's site generator should be **Hugo in spirit, but
tooled for software development** — a single static site that "hosts" the whole
project for a distributed dev model:

- **Issues** (the tracker) — built (ADR-0027/0030/etc.).
- **Docs** — render the repo's Markdown documentation. Not built.
- **Source tree** — a gitweb-style browser of commits/trees/files. Not built.

with: a **themeable template** layer so the output can be restyled; **simple
config** that enables/disables each section and carries per-language options
(syntax highlighting, doc layout); and all of it **static, offline, host-neutral**
(ADR-0029). This is explicitly the *developer portal* — issues + docs + code —
NOT the project's marketing/primary website, for which the stakeholder still
expects separate tooling such as Hugo.

The current `internal/web` generator hardcodes one base template and three
tracker sections (index/issues/board/track). That is section #1 of the portal;
it must become one of several config-selected sections sharing a themeable shell.

## Decision
gitrack's `web` generator is a **multi-section developer-portal static site
generator**. Architecture:

1. **Sections** are independent renderers, each producing part of the site:
   - `issues` (the tracker: index, issues, board, per-track) — EXISTS; reframed
     as the first section.
   - `docs` — renders repo Markdown (e.g. `docs/`, `README`) to HTML — future.
   - `source` — gitweb-style commit/tree/blob browser — future.
   Sections are enabled/disabled by config; a disabled section emits nothing and
   drops its nav entry.

2. **Theme / template layer.** The shared page shell (layout, nav, CSS) is a
   **theme**: a named set of templates + assets. A built-in default theme ships
   inline (offline, ADR-0029); a repo may override it from a theme directory.
   Sections render their body into the theme's shell, so restyling is one place.
   The current hardcoded `baseTmpl` becomes the default theme's shell.

3. **Config** lives under `.gitrack/config/site/` (plain files, the same
   guardrails-not-enforcement model, ADR-0026):
   - `sections` — enabled sections, one per line (default: `issues`).
   - `theme` — theme name/path (default: built-in).
   - `lang/<ext>` — per-language options for the source/docs sections
     (e.g. highlight mode), future.
   Absent config = sensible defaults (issues on, built-in theme), so existing
   behavior is unchanged until a project opts into more.

4. **Output stays static/offline/host-neutral** (ADR-0029): a self-contained
   directory deployable by file copy / aws s3 sync / Pages, openable from
   `file://`. The portal is the dev site; the project's primary site remains
   separate tooling (Hugo), by design.

This ADR is the architecture; the sections and theme/config system are built
incrementally against it. It reframes the shipped tracker web output as the
`issues` section and refactors `internal/web` toward a section + theme model.

## Consequences
- Easier: a coherent home for docs and source-tree rendering, and a single theme
  to restyle the whole portal; projects toggle sections with one config file.
- Easier: the existing issues/board/track output is preserved as a section; no
  behavior change by default.
- Harder: `internal/web` needs a refactor from hardcoded sections to a
  section+theme model; done incrementally so each step stays green.
- Harder: a theme-override system adds a (still offline) asset-resolution path;
  the default theme stays inline so `file://` works with no theme dir.
- Cost accepted: this is a larger subsystem than a single feature; it is built
  section by section (docs, then source) rather than all at once.

## Verification
- [ ] `docs/VISION.md` states the dev-portal (issues+docs+source), themeable,
      config-sectioned, not-the-marketing-site goal.
- [ ] The tracker output is produced by an `issues` section; with no site config
      the generated site is unchanged from today (issues on, default theme).
- [ ] `.gitrack/config/site/sections` enables/disables sections; a disabled
      section emits nothing and leaves no nav entry.
- [ ] A theme provides the shared shell; the default theme is inline (offline).
- [ ] docs and source sections, when added, plug in as sections under this model.
