# ADR-0048: Pages are pure data; presentation lives in swappable theme assets

## Status
accepted. Supersedes the "inline CSS" decision in ADR-0029 and the inline script in
ADR-0047.

## Context
The generated site put its whole stylesheet in a `<style>` block on every page, and
the copy button and board graph as inline scripts. Presentation was also in the
markup: bold tags, rules, `&middot;` between facts, arrows and dashes typed into
text, and a class called `muted`. An audit of a site with every section on found an
inline style on 72 of 72 pages, an inline script on 33, `&middot;` 43 times, and no
`<header>`, no `aria-current`, no `<time>` and no `data-*` anywhere. The only
navigation was bare links.

The result could not be restyled without editing Go, and a page could not be read
sensibly with CSS off. The aim, as with CSS Zen Garden or a Hugo theme, is for the
emitted pages to be content and structure, with a theme changing the look entirely.

## Decision

### The output contract
Every generated page:
1. has no `<style>`, no `style=` attribute, no inline script, no event handler, and
   no presentational element (`<b>`, `<i>`, `<hr>`, ...);
2. has no decoration in its text: no arrows, dashes, bullets or `&middot;`
   separators. Separators, arrows on back links, brackets and signs are drawn by the
   stylesheet with `::before` and `::after`;
3. has the landmarks `header`, `nav`, `main` and `footer`, with navigation as a list
   and `aria-current="page"` on the page's own entry; exactly one `main` and one
   `h1`;
4. exposes styling hooks that say what a thing is, never how it looks: `data-section`
   on `body` and on each nav entry, `data-state`, `data-status` and `data-kind` for
   values, role classes (`meta`, `empty`, `note`, `count`, `sha`, ...), and
   `<time datetime>` for dates;
5. reads in order and is usable with CSS and JavaScript both off.

### Theme assets
All presentation is in files under `assets/`, linked by a relative path (which works
from `file://` and any host):
- `assets/theme/` holds the chosen theme: `theme.css` (required), an optional
  `theme.js`, and whatever they refer to, keeping their layout;
- `assets/highlight.css` holds the code colors, written only when a page is
  highlighted, and chosen by the `highlight` setting, not by the theme;
- `assets/gitrack.js` is the copy button and `assets/board.js` is the board graph
  loader. They are behavior, not a theme, so swapping a theme never changes them.

Pages that need none of these link none of them.

### Themes
A theme is a directory. Built-in themes are embedded in the binary
(`internal/web/themes/<name>/`); a project's own are
`.gitrack/config/site/themes/<name>/`. A project theme wins over a built-in one of
the same name. The name comes from `.gitrack/config/site/theme`; an absent, empty,
invalid or unknown name falls back to `default` so a typo never breaks a build. A
project theme directory without `theme.css` is an error, because the directory is
there and the mistake is real.

A name must match `^[A-Za-z0-9][A-Za-z0-9._-]*$`, since it becomes a directory name.
Reading a project theme skips symbolic links, so a theme cannot pull files from
elsewhere into the site, and is capped at 200 files and 8 MiB.

The page shell is core, not part of a theme. The old idea that a theme supplies the
HTML shell is dropped: a theme that could change the markup would break the contract
and make the HTML differ between themes.

### Writing the files
Generation overwrites only the files it knows it owns and never removes `assets/` or
anything else. `--out` may be a repository root, and clearing it would be
destructive.

### The board script is a classic script
`assets/board.js` loads Mermaid with a dynamic `import()`. A `<script type="module">`
is refused when a page is opened from `file://`, which would break the graph there.

### Not covered
The `serve` preview is a separate small renderer and does not follow this contract
yet. Theme packaging and distribution, and selecting a theme per section, are not
addressed.

## Consequences
- Easier: restyling the whole site is one stylesheet. Another theme needs no Go. The
  pages read well without CSS, and a strict content-security policy that allows
  scripts from the site itself works unchanged.
- Easier: the contract is machine-checked, so presentation cannot creep back into
  the markup unnoticed.
- Cost accepted: a site is no longer a few self-contained HTML files. A page copied
  out on its own loses its styling, because the stylesheet is a separate file. The
  directory is still plain files and still works from `file://`.
- Cost accepted: one more request per page for the stylesheet, cached after the
  first.
- Harder: the board graph and the copy button cannot be verified by Go tests. The
  copy button is covered by a real-browser test. The Mermaid graph needs the network
  and was not verified in a browser.
- Risk: the token stylesheet is chosen apart from the theme, so a dark theme with a
  light code style shows light code blocks. The themes guide says to pair them.

## Verification
- [ ] No page of a site that uses every section matches any pattern the contract
      forbids.
- [ ] Every page has the landmarks, exactly one `main` and one `h1`, and its own nav
      entry marked current.
- [ ] Generating with each built-in theme gives byte-identical pages and a different
      `theme.css`.
- [ ] Every `href` or `src` that points into `assets/` resolves to a file that was
      written.
- [ ] A project theme beats a built-in one of the same name; an unknown or invalid
      name falls back to `default`; a theme without `theme.css` is an error; symbolic
      links are skipped; the file and size caps are enforced.
- [ ] Generating into a directory with an existing `assets/` leaves files the
      generator did not write.
- [ ] The copy button works in a real browser from `file://` with the script loaded
      as a file.
