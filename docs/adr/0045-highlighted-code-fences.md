# ADR-0045: Highlight fenced code blocks, under one site-wide setting

## Status
accepted

## Context
ADR-0043 added syntax highlighting to the source browser and left fenced code in
Markdown for later. Docs, blog posts, portal pages, and release notes are full of
fenced blocks (shell commands, YAML, Go), and they rendered as uncolored text next
to highlighted source pages.

The same chroma machinery applies, but two things change. Markdown pages are
produced by four different sections, so a page needs a way to say "I used
highlighting, attach the stylesheet" without every section knowing how. And the
setting was filed under `source/`, which stops making sense once it governs more
than the source browser.

## Decision

### Replace the block in the tree; leave everything else alone
A transformer finds each fenced code block whose info string names a language and
replaces it with a node holding finished HTML. Anything it does not replace, a
block with no language, an unknown language, or one that cannot be highlighted
within the limits, is left to goldmark's own renderer. So every fallback is
automatically the plain, escaped rendering with no extra code path. Replacing the
node avoids depending on the order in which goldmark applies competing renderers
for the same node kind.

### The language is what the author wrote
The first word of the info string is the language, looked up by name or alias
(`go`, `sh`, `bash`, `yaml`). Words after it (`go title=main.go`) are ignored. A
block with no language is never highlighted, and content is never sniffed: the
same rule as for source files (ADR-0043), and the same reason, a wrong guess is
worse than none. The language `text` and chroma's plain-text lexer count as no
language.

The output is `<pre class="chroma"><code class="language-go">...`, keeping the
`language-` class goldmark would have produced.

### The page carries the stylesheet only if it used highlighting
Rendering Markdown now returns the HTML and, separately, the token stylesheet,
which is empty unless at least one block was highlighted. Each section that
renders Markdown (portal, blog, docs, releases) passes that to the page, using the
optional stylesheet slot from ADR-0043. A page without highlighted code is
unchanged, so most pages carry no extra weight.

### One highlighter, one setting
The source browser and Markdown share a single highlighter for the whole
generation. That gives them one style and one total budget. The setting moves
from `.gitrack/config/site/source/highlight` to `.gitrack/config/site/highlight`:
`off`, or a chroma style name, default `github`. `off` therefore turns off
highlighting everywhere.

This is a deliberate break. The setting had existed for one commit and the project
has no tagged release, so there is nothing to keep compatible. The old location is
not read, and a test pins that, so a stale file cannot silently keep applying.

### Limits are the ones already in place
A block over 128 KiB, a block that takes too long to tokenize, and a generation
that has spent its 32 MiB budget all fall back to the plain rendering. Nothing
here needs a visible note: the plain block is correct, only uncolored. The total
budget is shared with the source browser, and sections render in a fixed order, so
the result stays deterministic.

### Not covered
- Inline code spans, which have no language.
- Code in issue descriptions and comments, which are not rendered as Markdown
  (ADR-0030).
- Line numbers, a copy button, or a language label on documentation blocks.
- Commit patches (ADR-0043).

## Consequences
- Easier: code in documentation reads like code, in any language chroma knows,
  with no script and no network, and it matches the source pages.
- Easier: no new dependency; this reuses what ADR-0043 added.
- Cost accepted, measured on gitrack's own docs (51 highlighted blocks across 14
  of 20 pages): generation time is unchanged within noise (about 50 ms either
  way), and the generated docs grow from about 280 KiB to about 340 KiB, a rise of
  roughly 21%, from token markup plus the stylesheet on each page that uses it.
- Harder: Markdown rendering takes a highlighter and returns a stylesheet, so a
  section that renders Markdown must pass the stylesheet through or its code
  renders uncolored. A test covers every section that does.
- Behavior change: the highlighting setting has a new location.

## Verification
- [ ] A fence naming a known language, including an alias such as `sh`, becomes a
      classed `chroma` block with its stylesheet; words after the language are
      ignored.
- [ ] A fence with no language, an unknown language, `text`, or with highlighting
      off keeps goldmark's default rendering and adds no stylesheet.
- [ ] Code containing HTML is escaped, never emitted as markup.
- [ ] Fences nested in lists and quotes are highlighted.
- [ ] A `#ref` inside a highlighted block is not linked.
- [ ] A block over the size cap, or beyond the total budget, stays plain; a skipped
      block spends no budget.
- [ ] Rendering with no highlighter is identical to the plain renderer.
- [ ] Headings still get ids alongside highlighted fences.
- [ ] Portal, blog, docs, and release-notes pages attach the stylesheet when they
      contain a highlighted block and carry none otherwise.
- [ ] The source browser and Markdown pages use the same stylesheet, a named style
      changes both, and `off` turns both off.
- [ ] A file at the old `source/highlight` location has no effect.
