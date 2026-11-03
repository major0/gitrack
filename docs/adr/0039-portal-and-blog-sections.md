# ADR-0039: Portal and blog sections (front page, about, contact, news)

## Status
accepted

## Context
ADR-0037 lists `portal` (front page, about, contact, news blog) as the first
non-tracker section of the full project website, and ADR-0038 provides the
shared offline Markdown renderer with portal-wide `#ref` autolinking. This ADR
specifies the `portal` and `blog` sections: where their content lives, how pages
are addressed, how the blog is ordered, and how the site **home page**
(`index.html`) is owned now that more than one section can exist.

Two concrete questions must be answered:

1. **Content source.** Marketing pages and blog posts are authored Markdown that
   lives *in the repo* (committed, versioned, portable by git — the north star
   of VISION.md), not generated from tracker objects.
2. **Home-page ownership.** The issues section writes the track index to
   `index.html`. The portal's front page also wants `index.html`. With several
   sections enabled, exactly one must own the site root.

## Decision

### Content lives under the site config dir, in the repo
- Portal pages: `.gitrack/config/site/portal/*.md` — each file is one page.
  `front.md` is special (the front page; see home ownership). Every other
  `*.md` becomes `portal/<name>.html` with a nav entry labeled from its title
  (front matter `title:`, else the first `# H1`, else the file name). Conventional
  pages `about.md` and `contact.md` need no special handling — they are just
  portal pages, and the stakeholder's "about/contact" are satisfied by shipping
  those files.
- Blog posts: `.gitrack/config/site/blog/*.md` — each file is one post, rendered
  to `blog/<slug>.html`, plus a `blog/index.html` listing posts newest-first.

Both are plain tracked files (ADR-0026 guardrails: absent dir ⇒ the section
emits nothing and drops its nav entry). All bodies render through
`renderMarkdown` (ADR-0038), so `#refs` autolink everywhere.

### Minimal front matter
A post/page may begin with a tiny YAML-ish front-matter block delimited by `---`
lines, parsed with a small key:value reader (no YAML dependency — keep the
dependency count at one, ADR-0038):
- `title:` — page/post title (fallback: first `# H1`, then the file base name).
- `date:` — post date `YYYY-MM-DD` (blog only; fallback: none → sorts last, and
  the post is flagged undated in output). Date is authored, not derived from git,
  so a post reads consistently offline and when copied (ADR-0029). (A later ADR
  may add git-derived dates; authored keeps slice 1 self-contained.)
Front matter is optional; a bare Markdown file is a valid page/post.

### Blog ordering and index
`blog/index.html` lists posts by `date` descending (newest first); undated posts
sort last by file name. Each entry shows the title, date, and links to the post
page. This reverse-chronological Markdown list is exactly the shape a `releases`
section and any future feed reuse (ADR-0037 "very minor hop").

### Home-page ownership: the first enabled section owns `index.html`
The site root `index.html` is written by the **first enabled section** (config
order). The section framework tells each section whether it is the home section:
- issues-only (default): issues is first ⇒ it writes the track list to
  `index.html` exactly as today (Phase 0 byte-identical guarantee preserved).
- portal enabled and first (the ADR-0037 catalog order puts portal first): portal
  writes `front.md` to `index.html`; the issues section, no longer home, writes
  its track list to `tracks.html` and its "tracks" nav entry points there.
A section that is not the home section never writes `index.html`. This keeps a
single, order-driven rule rather than per-section special cases, and no two
sections ever race for the root.

### Nav
- portal contributes: `home` → `index.html` (only when it is the home section),
  then one entry per non-front portal page, then `blog` → `blog/index.html` when
  the blog section is enabled and has posts.
- The blog is its own section (`blog`) so it can be enabled independently; when
  both are on, portal's nav shows the blog link for a single coherent bar. When
  only `blog` is enabled, the blog section contributes its own `blog` nav entry.

### Theme CSS for Markdown
The default theme gains styles for Markdown block elements (headings in body,
blockquotes, tables, post meta) so portal/blog/docs/releases render legibly.
This is additive to the inline CSS; it does not change tracker pages' existing
look beyond shared element defaults, and stays inline/offline (ADR-0029).

## Consequences
- Easier: a project authors its marketing pages and news posts as repo Markdown;
  they render into the same themed, autolinked, offline site as issues.
- Easier: one order-driven home-ownership rule; issues-only output is unchanged,
  and enabling portal cleanly relocates the track index to `tracks.html`.
- Easier: the reverse-chronological blog index is the template the releases
  section and feeds reuse.
- Harder: the section framework must expose "am I the home section?" and the
  issues section must switch its index path accordingly; done behind the same
  default so Phase 0 behavior is preserved.
- Cost accepted: a minimal hand-rolled front-matter reader (no YAML dep); it
  handles `key: value` lines only, which is all pages/posts need now.

## Verification
- [ ] With `portal` enabled, `.gitrack/config/site/portal/front.md` renders to
      `index.html`, other `portal/*.md` to `portal/<name>.html` with nav entries,
      and the issues track index moves to `tracks.html`.
- [ ] With `blog` enabled, `.gitrack/config/site/blog/*.md` render to
      `blog/<slug>.html` and `blog/index.html` lists them newest-first by `date`.
- [ ] Front matter `title:`/`date:` are honored; a bare Markdown file still works
      (title falls back to H1/filename; undated posts sort last).
- [ ] `#refs` in portal/blog bodies autolink (ADR-0038); code/links are skipped.
- [ ] Issues-only tracker output is structurally unchanged from Phase 0: the
      nav, body, and all tracker content are byte-identical; the only diff is the
      additive Markdown block CSS in the shared theme `<style>`, which matches no
      element the tracker pages render (so their appearance is unchanged). The
      home rule is a no-op when issues is the first/only section.
- [ ] Absent `portal/`/`blog/` dirs ⇒ the section emits nothing and leaves no nav
      entry (ADR-0026).
