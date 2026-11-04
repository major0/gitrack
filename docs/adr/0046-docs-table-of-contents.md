# ADR-0046: A table of contents on long docs pages

## Status
accepted

## Context
Docs pages can run to dozens of sections. With heading ids and permalinks in place
(ADR-0044) a reader can link to a section, but still has to scroll to find one.
The docs sidebar (ADR-0040) lists pages, not the sections within the current page.

"Long" has to mean something concrete, and showing a contents list on a page with
three short sections is clutter. The published docs themselves give the data: their
twenty pages have between 2 and 33 h2/h3 headings, and most guides sit at 5 to 7.

## Decision

### What a page lists
The table of contents lists the page's `##` and `###` headings, in document order,
as links to their ids. An `###` nests under the `##` before it; one with no `##`
before it is listed at the top level rather than dropped.

Not listed: the `#` heading (it is the page title), `####` and deeper (too fine to
navigate by), and any heading that is not a direct child of the page, such as one
inside a block quote or a list, or text inside a code fence that merely looks like a
heading. Those still get ids and permalinks. They simply are not page structure.

Entries are the heading's visible words, as plain escaped text: no code styling, no
link destinations, and never the "Permalink" label added to the heading.

### When it appears
A page gets a table of contents when it has at least **8** such headings. That
threshold is read off the published docs: it selects six of twenty pages
(`tracks`, `source`, `website`, `configuration`, the format spec, and the command
reference), which are the ones a reader scrolls through looking for a section. A
threshold of 4 would have given nearly every page one, including pages short enough
to read at a glance.

It is configurable, because "long" is a judgment. `.gitrack/config/site/docs/toc` is
one line: `off` or `0` disables it, a positive number sets the threshold, and
anything else, or no file, uses the default. A bad value never fails generation
(ADR-0026).

### Where it goes
Immediately before the first listed heading, so the title and any introduction come
first and the contents introduce the sections. It is a labelled `<nav>` landmark,
"On this page", styled as a small boxed list. It is plain HTML and CSS: no script
and no collapsing widget.

### Docs pages only
Only the docs section asks for one. Blog posts, portal pages, and release notes are
rendered without it. The renderer takes the threshold as an option that defaults to
off, so another section can opt in later without changing this one.

### How it is built
The contents are produced in the same pass that assigns heading ids, which already
holds each heading's final id (with duplicates numbered) and its visible text from
before the permalink is added. They are inserted as a block of finished HTML, using
the same raw-block node that highlighted code fences use (ADR-0045), now shared
rather than duplicated. Because the block is built from the ids actually assigned,
its links cannot disagree with the headings.

## Consequences
- Easier: a reader can see a long page's structure and jump to a section, with no
  script, and the links are the same ids the permalinks use.
- Easier: the existing link checker over the published docs now covers these links
  too, so a contents link can never silently point at a missing id.
- Cost accepted: the page grows by a short list, and a page with many `###`
  headings gets a long contents list. The command reference lists 33 entries; that
  is the page where jumping straight to a command matters most.
- Harder: nothing new to maintain beyond one more setting. Ids are text-derived, so
  renaming a heading changes its link (ADR-0044); that applies to these links
  equally.
- Not done: a sticky or side-column contents list, a collapsible one, per-page
  opt-out, and contents for blog or release pages.

## Verification
- [ ] A page with at least the threshold of listed headings gets exactly one table
      of contents; a page with one fewer gets none.
- [ ] It sits after the introduction and before the first listed heading, and does
      not list the page title.
- [ ] `##` and `###` are listed, `###` nests under its `##`, a `###` with no `##`
      before it is listed at the top level, and `####` is not listed.
- [ ] Headings inside quotes, lists, and code fences are neither listed nor counted
      toward the threshold, and still have ids.
- [ ] Every link is the id of the heading it names, in order, including numbered
      duplicates and percent-encoded non-ASCII ids.
- [ ] Labels are plain escaped text and never contain "Permalink".
- [ ] A heading with no visible text is labelled by its id.
- [ ] `off`, `0`, a number, a negative number, and garbage in the `toc` file behave
      as described.
- [ ] Portal, blog, and release-notes pages never get one.
- [ ] It coexists with permalinks and highlighted code fences, and links do not nest.
