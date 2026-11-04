# ADR-0047: A copy button on code blocks, as a small inline script

## Status
accepted. The script is no longer inline: ADR-0048 moved it to `assets/gitrack.js`,
loaded by a `<script src>` at the end of the body. The statements below about an
inline script, and about the button's styles living in the page, are superseded by
that. Everything else stands.

## Context
Documentation and source pages are full of code a reader wants to paste: install
commands, configuration, whole files. Selecting a block exactly, without picking up
a line number or a neighbouring paragraph, is fiddly, and on a phone it is worse.
A "Copy" button is the standard answer.

It is also the first feature that cannot be done in HTML and CSS. Every behavior so
far, highlighting (ADR-0043, ADR-0045), heading permalinks (ADR-0044), the table of
contents (ADR-0046), is generated markup with no script. Copying to the clipboard
needs JavaScript. The one script the site already had, Mermaid on the board page,
is an enhancement that is allowed to reach for the network (ADR-0029). This one
must not: it has to work offline, from `file://`, and on any dumb host.

## Decision

### An inline script, with no network and no library
A single self-contained script of about seventy lines (2 KB) is written into the page. It
makes no network request, loads nothing, and defines no globals. The page works
exactly as before without it.

### Progressive enhancement: the script creates the buttons
The buttons do not exist in the generated HTML. The script builds them. So a visitor
with JavaScript off, or a host whose content-security policy forbids inline scripts,
sees the page unchanged, with no dead control and nothing to explain. The
alternative, shipping buttons in the markup and hiding them until the script runs,
would leave a broken control for those visitors.

### Which blocks, and what is copied
A "code block" is a `<pre>` whose first child is `<code>` (every Markdown fence,
highlighted or not, and indented code) or a source page's numbered `<pre
class="blob">`. Everything else, the board graph, commit patches, commit messages,
and issue descriptions, is left alone.

The script wraps each block in a positioned `<div>` and puts the button beside the
block, not inside it. Copying reads the block's own text, so it can never include the
button's label. Source pages draw line numbers with CSS, so they are not part of the
text either, and a copied file is exactly the file.

### Which pages carry the script
Only pages that contain a code block, and only if the button is not turned off. The
renderer reports whether its output holds a code block, by looking for the two
strings only our own renderers produce: goldmark's `<pre><code` and our
`<pre class="chroma">`. Authored text cannot forge them, because raw HTML in
Markdown is disabled and text is escaped. The stylesheet for the button lives in the
theme, a few hundred bytes on every page, so that a page's stylesheet stays a
function of its highlighting alone.

### Setting
`.gitrack/config/site/copy`: `off` leaves the script out of every page. Anything
else, or no file, leaves it on. A project that wants a site with no script of its
own (Mermaid aside) has a way to say so.

### Copying
The asynchronous clipboard API is used where it exists. When it is missing, or
refuses (it is withheld in some contexts), the script falls back to selecting a
hidden textarea and `document.execCommand('copy')`, which also works from
`file://`. If both fail the button says "Copy failed". The outcome is shown in the
button's label and announced through a polite live region, and focus stays on the
button.

### Accessibility
The button is a real `<button type="button">` with an accessible name, reachable by
keyboard, and it appears on hover or on focus. Where there is no hover (touch) it is
always shown, faded. It cannot submit a form.

### Not covered
Per-block labels or filenames, a "copy as" menu, copying a selection, and a button on
commit patches.

## Consequences
- Easier: a reader can paste a command or a whole file in one press, including on a
  phone, and the copy is exact.
- Cost accepted: gitrack now ships an inline script on most documentation and source
  pages. It is small, local, and optional, but it is a script, and the earlier
  statement that only the board graph is an enhancement is no longer true. The user
  guides were corrected to say so.
- Cost accepted: about 0.5 KB of stylesheet on every page, and about 2 KB of script
  on pages with code.
- Harder: this behavior cannot be covered by Go tests. It is covered by a real-browser
  test, `tests/copy-button.sh`, which drives headless Chrome and skips itself where
  Node or Chrome is missing. That test replaces the clipboard with a recorder, because
  a programmatic click carries no user gesture, so it proves the script picks the
  right blocks, passes the right text, takes the fallback path, and reports the
  outcome, but not that an operating system's clipboard receives the text.
- Risk: a host with a strict content-security policy silently gets no buttons. That
  is the intended failure mode, but nothing tells the site's owner.

## Verification
- [ ] In a real browser, every Markdown code block and every source page gets one
      button, placed beside its block and not inside it, with an accessible name and
      type `button`.
- [ ] The copied text is exactly the code: indentation, tabs, and escaped characters
      intact, no line numbers, and no button label.
- [ ] When the clipboard API is missing, or refuses, the fallback copies the same
      text and leaves no temporary element behind.
- [ ] The label and the live region report success, focus stays on the button, and
      the label resets.
- [ ] The script picks only code blocks even on a page that also holds other `<pre>`
      elements.
- [ ] Pages without code blocks, and every page when the setting is `off`, carry no
      script, and the board graph, track descriptions, and commit patches get no
      button.
- [ ] Authored text cannot make a page claim to have a code block.
- [ ] The script contains no URL, no network call, and no closing script tag.
- [ ] A page without the script ends exactly as it did before this feature existed.
