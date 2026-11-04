# ADR-0044: Rendered headings get GitHub-compatible ids and a permalink

## Status
accepted

## Context
Headings in rendered Markdown (docs, portal, blog, release notes) had no ids, so
a link to a section, `page.md#section` or just `#section`, landed at the top of
the page. The docs section rewrites `page.md` links to `page.html` and preserves
the fragment (ADR-0040), but the fragment had nothing to match. In practice that
meant an author had to drop a section link to avoid shipping a dead anchor.

Authors write fragment links for how GitHub renders their Markdown, because that
is where the same files are read in the repository. For a link to work in both
places the ids must be the ones GitHub generates, not merely unique ones.

## Decision

### Our own slug rules, matching GitHub
goldmark ships a heading-id option, but it is not GitHub-compatible: it discards
every non-ASCII character (an accented or CJK heading collapses to `heading`) and
rewrites `_` to `-`. So the section writes its own generator, in the manner of
GitHub's:
- lower-case the text;
- keep letters, combining marks, numbers, `_` and `-`;
- turn each space into `-`;
- drop everything else (punctuation, symbols, emoji).

`snake_case names` is `snake_case-names`; `Café au lait` is `café-au-lait`;
`C++ & Go` is `c--go`. A heading with nothing left (only punctuation) is
`section`.

### Ids come from the visible text
The id is built from the heading's text nodes after parsing, not from its source
line, so code spans, emphasis, and the destination of a link inside a heading do
not leak into it. The text is read before the autolinker rewrites any `#ref`
inside the heading, so ids come from what the author wrote.

### Duplicates are numbered per document
A repeated heading gets `-1`, `-2`, ... appended, counted separately for each base
slug, and numbering restarts for every document. A heading that slugs to an
already-taken name, such as one titled `Intro 1` after two `Intro` headings, is
itself numbered rather than colliding.

### A permalink in each heading
A link to `#id` is appended inside every heading so a reader can copy a link to
the section. It is hidden until the heading is hovered or the link is focused, and
is always shown at reduced strength on devices without hover.

Accessibility: the link's text is the word "Permalink", so it has an accessible
name, and it has a `title`. CSS hides the word and draws a pilcrow in its place
(marked decorative where the browser supports that). goldmark does not allow
`aria-label` on a rendered link, so the name has to come from the text. The cost
is that a screen reader announces a heading as its text followed by "Permalink".

### Safety
An id can only contain letters, combining marks, numbers, `_` and `-`, so no text
in a heading can add a quote or angle bracket to the attribute. Raw HTML in a
heading stays inert as everywhere else in the renderer (ADR-0038).

## Consequences
- Easier: `page.md#section` and `#section` links work, on the site and on GitHub,
  including for non-English headings.
- Easier: the docs tree can be checked mechanically. A test renders the real docs
  and fails if any `page#fragment` link points at an id that does not exist, so
  renaming a heading cannot silently break a link.
- Harder: ids now depend on heading text, so renaming a heading changes its id and
  breaks links to the old one. That is inherent to text-derived ids, and the link
  checker is what catches it.
- Cost accepted: every rendered heading carries an extra link, and the stylesheet
  grows by a few rules.
- Not done: ids for headings in the issue and comment text, which are not rendered
  as Markdown (ADR-0030), and a table of contents.

## Verification
- [ ] Every heading in rendered Markdown has an id, including setext headings and
      headings inside lists and quotes.
- [ ] Ids follow the GitHub rules, including `_`, non-ASCII letters, and
      punctuation-only headings.
- [ ] A duplicate heading is numbered, a heading that collides with a generated id
      is itself numbered, and numbering restarts for each document.
- [ ] Inline code, emphasis, and links in a heading do not change its id beyond
      their visible text.
- [ ] A `#ref` in a heading still links, and the permalink does not nest inside it.
- [ ] A hostile heading cannot put anything but safe characters in an id.
- [ ] `other.md#frag` becomes `other.html#frag`, and `#frag` resolves on the same
      page.
- [ ] Every `page#fragment` link in the published docs resolves to an existing id.
