# ADR-0038: Shared offline Markdown rendering + portal-wide `#ref` autolinking

## Status
accepted

## Context
ADR-0037 makes `git track web` a full project website whose sections (portal,
blog, docs, release notes, and API prose) render **Markdown**, and requires that
`#ref` autolinking (ADR-0011/0030/0034) work across every section that renders
authored text — not just issue descriptions and comments.

Two decisions were deferred to here:

1. **Markdown rendering.** ADR-0030 explicitly deferred "a full Markdown parse."
   The tracker's `autolink` currently runs on raw text (escape-then-linkify, no
   block structure). Docs/blog/release notes are real Markdown (headings, lists,
   code fences, links), so the site now needs a Markdown→HTML renderer. It must
   honor ADR-0029: pure-Go, offline, no CDN/JS, CSS supplied inline by the theme.

2. **Autolink composition.** Once content is Markdown, naive regex-on-rendered-
   HTML linkifying is wrong: it would turn `#` inside code spans/fences,
   heading anchors, and URLs into bogus links, and risks malforming the HTML the
   Markdown renderer already escaped. Autolinking must compose *with* the
   Markdown parse, touching only prose text.

## Decision

### Markdown library: goldmark (pinned)
Use **`github.com/yuin/goldmark`** (pinned in `go.mod`) as the single Markdown
renderer for every Markdown section. Rationale:
- Pure Go, zero cgo, no network — satisfies ADR-0029 offline/host-neutral.
- CommonMark-compliant with a real AST (not struct-based), so autolinking can be
  an AST transform rather than HTML post-processing.
- MIT-licensed (compatible with gitrack's MIT license); it is the renderer Hugo
  itself uses, so it is well-maintained and battle-tested.
- It is gitrack's **first external dependency**. We accept exactly one dependency
  (plus its own, which goldmark keeps to the standard library) for a capability
  that is unreasonable to hand-roll correctly (CommonMark is large and subtle).

goldmark is configured with: GitHub-Flavored-Markdown extension (tables,
strikethrough, autolinked bare URLs, task lists — matches how developers write),
`WithXHTML`/`WithHardWraps` left at CommonMark defaults, and **no** raw-HTML
passthrough by default for untrusted content (see Safety).

### Autolinking is an AST transform, shared by one renderer
`#ref` autolinking is implemented as a goldmark **AST transformer** that runs
after parsing and before rendering. It walks the document and, **only inside
`ast.Text` nodes that are not within a code span, code block, autolink, or
existing link**, finds `#<handle>` / cross-repo `#<url>?track=<uuid>` tokens and
replaces each resolvable one with a link AST node:
- local handle → relative link to the target's page via `core.Resolve` (track
  today; extended to release/doc targets as those sections land, ADR-0037 §2);
- cross-repo → external link, no fetch (ADR-0034);
- non-resolving token → left as plain text (ADR-0030's resolve-gated rule).

Because the transform operates on the AST, goldmark's own escaping and the
skip-code/skip-link rules fall out for free — no regex over generated HTML, no
double-escaping. The token charset and resolution reuse the exact rules already
in `web.autolink` (ADR-0011/0030/0034); that logic is factored so the raw-text
path (issue descriptions/comments) and the Markdown path share one tokenizer and
one resolver.

### One entry point
The web package exposes a single `renderMarkdown(repo, root, src) template.HTML`
used by every Markdown section. It builds a goldmark instance with the autolink
transformer bound to `repo`/`root` and returns safe HTML for the theme shell.
The existing raw-text `autolink` (for non-Markdown fields) stays, now sharing the
tokenizer/resolver with the AST transform.

### Safety (ADR-0029/0030 carried forward)
- Untrusted Markdown is rendered with raw inline/block **HTML disabled** (goldmark
  default), so an embedded `<script>`/`<iframe>` in authored content is rendered
  as text, not markup. (A future trusted-content opt-in, e.g. the project's own
  portal pages, can relax this per-section, but the default is safe.)
- Output stays fully static/offline: no JS, no external CSS; syntax highlighting,
  if added later, must be build-time HTML+inline-CSS, never a CDN script
  (ADR-0029), and is out of scope for this ADR.

## Consequences
- Easier: real Markdown (headings, lists, tables, code) renders consistently in
  docs/blog/releases/portal, and `#refs` link the same way everywhere, from one
  renderer and one resolver.
- Easier: AST-based autolinking is correct-by-construction around code and links,
  retiring the "token-level, not a full parse" limitation of ADR-0030 for
  Markdown sections (the raw-text path keeps its conservative behavior).
- Cost accepted: gitrack takes its first third-party dependency (goldmark, MIT,
  pure Go). We judge the CommonMark-correctness win worth the one dependency and
  pin the version.
- Harder: the autolink tokenizer/resolver must be factored out of the current
  `autolink` so both paths share it; done without changing raw-text behavior.
- Deferred: syntax highlighting, raw-HTML opt-in for trusted portal pages, and
  release/doc link targets land with their own sections/ADRs.

## Verification
- [ ] `go.mod` pins `github.com/yuin/goldmark`; `go build`/`vet`/`test`/
      `staticcheck` stay green and the binary stays single-file, offline.
- [ ] `renderMarkdown` turns a Markdown string (headings, list, code fence,
      link) into HTML with no network reference and no external stylesheet.
- [ ] A resolvable `#<handle>` in Markdown **prose** becomes a relative link; the
      same token inside a code span or fenced block is left untouched.
- [ ] A cross-repo `#<url>?track=<uuid>` in Markdown becomes an external link (no
      fetch); a non-resolving `#token` stays plain text.
- [ ] Authored raw HTML (`<script>`) in untrusted Markdown is escaped, not
      executed.
- [ ] The raw-text `autolink` path (issue descriptions/comments) is unchanged
      (existing tests pass) and now shares the tokenizer/resolver with Markdown.
