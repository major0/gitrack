# ADR-0041: Releases are tracks of kind `release`, shown by a `releases` section

## Status
accepted

## Context
ADR-0037 lists `releases` as a section: release notes per tag that link the
issues they shipped. It deferred the data model to a dedicated ADR ("a `release`
object kind: tag + date-from-git + Markdown notes + closes-refs").

gitrack already has one universal object (ADR-0002) whose `kind` is cosmetic
(ADR-0006), a Markdown `description`, a UUID and slug, comments, and forward-only
relationships (ADR-0005). A release needs exactly those things, so inventing a
second object type or a parallel store would contradict "smart data, dumb code".
Two things are genuinely new: a way to say *which tracks a release shipped*, and
a *date* that is not hand-typed.

## Decision

### A release is a track with `kind = release`
No new object type. `git track create -k release "v1.2.0"` makes one, and every
existing verb (`show`, `update`, `comment`, `close`, `get`, `set`) already works.
Its fields reuse the track layout:

- `title` — the version or name (`v1.2.0`); the slug derives from it as usual.
- `description` — the release notes, Markdown, rendered with `renderMarkdown`
  (ADR-0038) so `#refs` autolink.
- `state` — `open` means *upcoming*; `closed` means *released*. This is the
  fixed axis (ADR-0019), not a new field.
- `tag` — optional single-valued field: the git tag this release corresponds to.
  When absent the title is used as the tag name. Free-form, so `set <ref>.tag
  v1.2.0` writes it; there is no dedicated command.

### What a release shipped: the `ships` relationship
A new forward relationship directory, `ships/<target-uuid>`, on the release:
"this release shipped that track". It follows ADR-0005: only the forward link is
stored, the inverse ("shipped in") is computed. It is written with a
relation-named verb like the others: `git track ships <release> <target>`, and
removed with `unlink <release> ships <target>`.

`ships` has **no scheduling effect**. The board and `Blockers` read only
`depends-on` and `child-of`, so shipping a track never blocks or unblocks
anything. It is a record, not a constraint.

Shipping is explicit rather than inferred from commit messages. Inference would
need a window of history per release, and ADR-0031/0033 already settled that
gitrack does not guess state from passive observation.

### The date is derived from git, not stored
A release's date is the committer date of its tag's commit, read with
`git log -1 --format=%cI refs/tags/<tag>^{commit}`. The `refs/tags/` prefix means
a tag value can never be parsed as a git option. If the tag does not exist (an
upcoming release, or a repo without that tag) the date falls back to the track's
`created` timestamp. This mirrors comment author/date (ADR-0023): git already
knows, so we do not store a second copy that could disagree.

### Guardrails, not enforcement (ADR-0026)
`ships` is accepted from any track, and a non-release may be shipped. Whether a
track is "really" a release is the cosmetic `kind`; the generator simply lists
tracks whose kind is `release`.

### The `releases` section
An opt-in section (`releases` in `.gitrack/config/site/sections`) that renders
from the tracker, not from site content files:

- `releases/index.html` — every release newest-first by derived date, each row
  showing title, date, whether it is upcoming or released, and how many tracks it
  shipped.
- `releases/<slug>.html` — one release: title, date, tag, the notes rendered as
  Markdown, and the shipped tracks (title, state) linking to their track pages.
- Top-bar nav: one `releases` entry, only when at least one release exists.

With the `issues` section disabled the track pages do not exist, so shipped
tracks render as plain text instead of dead links.

### Shipped-in on track pages
Because `ships` is forward-only, the issues section computes the inverse once per
generation (a map from shipped track to the releases that ship it) and shows
"Shipped in: v1.2.0" on each shipped track's page. One pass over the tracks, not
a scan per page.

### Autolinking
A release is a track, so `#v1-2-0` already resolves through `core.Resolve` to its
track page (ADR-0030). Nothing special is needed in the resolver, which is why the
ADR-0037 plan to extend resolution to release targets costs nothing here.

## Consequences
- Easier: no new object type, store, or storage code. Releases get comments,
  slugs, aliases, history, and autolinking for free.
- Easier: the date is always right, because it is read from the tag.
- Easier: the reverse-chronological Markdown list is the same shape as the blog
  (ADR-0039), so a feed or changelog export later is a small step.
- Harder: `ships` adds a fifth relationship kind to document, and the tracks
  index and issues view will list release tracks alongside other tracks. We
  accept that: it is the same consequence as any other `kind`.
- Cost accepted: a release's shipped set is maintained by hand with `ships`.
  There is no "everything closed since the last tag" automation; that would be a
  separate, opt-in command.

## Verification
- [ ] `git track ships <release> <target>` writes `ships/<target-uuid>` on the
      release and `unlink <release> ships <target>` removes it.
- [ ] `ships` does not change the board: a track shipped by an open release is
      still Ready/Blocked exactly as before.
- [ ] A release's date comes from its tag's commit when the tag exists, and from
      `created` otherwise; a tag name beginning with `-` is not treated as an
      option.
- [ ] With `releases` enabled and at least one `kind=release` track, the site has
      `releases/index.html` (newest-first) and `releases/<slug>.html` with the
      notes as Markdown, and a `releases` nav entry.
- [ ] No release tracks ⇒ no `releases/` output and no nav entry.
- [ ] A shipped track's page shows "Shipped in" with a link to the release.
- [ ] With `issues` disabled, shipped tracks on a release page are plain text.
