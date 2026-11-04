# ADR-0043: Syntax highlighting in the source browser, at build time, with chroma

## Status
accepted

## Context
ADR-0042 shipped the source browser with plain escaped text and left highlighting
for later, with one constraint carried from ADR-0038: it must be build-time HTML
with inline CSS and no script, so the site stays offline and host-neutral
(ADR-0029). A highlighter written for this project would have to cover the
languages a real repository contains, which is hundreds of grammars; that is not
reasonable to hand-roll or maintain.

## Decision

### Use chroma, at generation time
Highlighting uses `github.com/alecthomas/chroma/v2` (MIT, pure Go), with its one
transitive dependency `github.com/dlclark/regexp2/v2` (MIT). It runs while the
site is generated and emits classed `<span>` elements plus a stylesheet. Nothing
runs in the browser. This is gitrack's second third-party dependency, accepted for
the same reason as the first (ADR-0038): the capability is unreasonable to
reimplement.

Chroma is the highlighter Hugo's Markdown stack uses, supports about 280
languages, and ships roughly 70 color styles.

### Classes and one stylesheet, only where used
Tokens get CSS classes, and the style's rules are emitted once per page in a
second `<style>` block after the theme's own. Only pages that were actually
highlighted carry it. The tree, log, commit, and plain-text pages are unchanged.
The shared theme gains an optional slot for such a stylesheet, written so a page
without one is byte-identical to before.

### Line structure is preserved
Each source line is still its own numbered span (ADR-0042). Chroma's tokens can
span lines (block comments, multi-line strings), so tokens are split per line
before formatting and every line's tags balance on their own. A trailing newline
inside a token is stripped so it cannot add a blank row. If the highlighted
result does not have the same number of lines as the source, the file falls back
to plain text rather than shipping misnumbered code.

The line span's class is `src-line`. It was `l`, which collides with chroma: that
is the class chroma uses for "Literal" tokens, 37 of its 74 styles define it, and
every literal token would also have incremented the line counter. `ln` collides
the same way for 14 styles.

### Language by file name only
The language comes from the file name (extension or well-known name such as
`Makefile`). Content is not sniffed: guessing a language from text labels plain
files wrongly, and a wrong guess is worse than none. chroma's plain-text lexer
(what `*.txt` maps to) is treated as "no language", since it has nothing to color
and would only attach a stylesheet to an ordinary page.

### Configuration
One optional single-line file, `.gitrack/config/site/source/highlight`:
- absent: the `github` style (light);
- `off`: no highlighting at all;
- any chroma style name (`monokai`, `dracula`, `solarized-light`, ...): that style.
An unknown name falls back to `github` rather than failing (ADR-0026).

### Bounded cost
Chroma is regex-based and not fast: measured at roughly 300 KB/s. A hostile or
very large input must not stall generation, and a very large repository must not
silently take minutes, so three limits apply, each ending in plain text and a
visible note rather than an error:
- a file over 128 KiB is not highlighted;
- tokenizing a file that runs past 2 seconds is abandoned (chroma also puts a
  250 ms timeout on each regex match);
- one generation highlights at most 32 MiB of source in total. This is a byte
  count, not a clock, and files are visited in git's fixed path order, so the same
  repository always highlights exactly the same files regardless of machine speed.

### Not covered
- Commit patches are not highlighted: a patch mixes languages and shows partial
  lines, which lexers cannot parse reliably.
- Fenced code in Markdown (docs, blog, release notes) is not highlighted by this
  change; that is the Markdown renderer's concern.
- Content-based language detection, per-language style overrides, and a dark-mode
  stylesheet for the same page are not provided.

## Consequences
- Easier: source pages are readable as code, in any of the languages a project
  uses, with no script and no network.
- Cost accepted, measured on gitrack's own repository:
  - the binary grows from 14.1 MB to 18.6 MB (+4.3 MiB, +32%), because the
    lexers are embedded;
  - generating the source section went from about 0.4 s to about 4.0 s;
  - the generated site grew from 3.7 MiB to 6.3 MiB (token markup plus a
    stylesheet of a few KiB on each highlighted page).
  `highlight` set to `off` removes the generation-time and output cost, though not
  the binary size.
- Harder: a second dependency to track. The three limits are the price of
  running a regex engine over untrusted file content.

## Verification
- [ ] A recognized file is rendered with classed token spans and its stylesheet;
      its page is marked `chroma`; the line count equals the plain rendering's.
- [ ] A block comment spanning several lines yields lines whose tags each balance.
- [ ] Source containing HTML (a `<script>` string) is escaped, never emitted as
      markup.
- [ ] An unknown extension, a `.txt` file, and `highlight = off` produce plain
      numbered lines and no stylesheet.
- [ ] A file name with no extension but a known name (`Makefile`) is recognized.
- [ ] A file over 128 KiB, an exhausted time budget, and an exhausted total budget
      each fall back to plain text with a note and keep every line.
- [ ] The total budget is deterministic: a file that does not fit is skipped
      without spending budget, and a smaller later file still highlights.
- [ ] Tree, log, commit, and plain-text pages carry no token stylesheet.
- [ ] An unknown style name produces the same output as the default.
- [ ] The line span class does not collide with any chroma class.
