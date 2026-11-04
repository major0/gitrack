# Releases

A release is a track like any other, with `kind` set to `release`. It carries
the release notes, remembers which tracks it shipped, and gets its date from your
git tag. The website's `releases` section turns these into a changelog.

## Create a release

```sh
git track create -k release "v1.2.0"
```

The title is the version. The track starts open, which means **upcoming**.
Write the release notes in its description, as Markdown:

```sh
git track set v1-2-0.description "## Highlights

- Faster board rendering
- Threaded comments in the web view

Thanks to everyone who filed issues. See #session-refresh for the big one."
```

`#refs` in the notes autolink to their tracks on the site.

## Record what it shipped

Use `ships` to link the release to each track it delivered:

```sh
git track ships v1-2-0 new-login-flow
git track ships v1-2-0 session-refresh
```

This is a record, nothing more. Shipping a track never blocks or unblocks
anything, so the [board](boards.md) is unaffected. Remove a link with
`git track unlink v1-2-0 ships session-refresh`.

You maintain this list yourself. gitrack does not guess which tracks belong to a
release from commit history, so the notes always say exactly what you intend.

## Tie it to a git tag

By default the release's title is also its tag name. If your tag is named
differently, set it:

```sh
git track set v1-2-0.tag release-1.2.0
```

## Where the date comes from

You never type a date. A release's date is the date of the commit its tag points
at. If the tag does not exist yet (an upcoming release), it falls back to the day
the release track was created. So tag the release, and its date on the site
becomes the real release date.

```sh
git tag v1.2.0
git track close v1-2-0       # mark it released
```

## Upcoming and released

`state` carries the meaning:

- **open** — upcoming. Listed as "upcoming" on the site.
- **closed** — released. Listed as "released".

Close the release when you cut it, the same as any other track.

## On the website

Enable the section in `.gitrack/config/site/sections`:

```
issues
releases
```

You get:

- `releases/index.html` — every release, newest first, with its date, whether it
  is upcoming or released, and how many tracks it shipped.
- `releases/<slug>.html` — one release: the notes rendered as Markdown, the tag,
  and the list of shipped tracks with their state, each linking to its track
  page.
- A `releases` entry in the top navigation, shown only once at least one release
  exists.

Each shipped track's own page gains a **shipped in** line linking back to the
release. With the `issues` section turned off there are no track pages, so the
shipped tracks appear as plain text rather than links.

A release is also a normal track, so it appears in the tracks index and issues
view, and `#v1-2-0` autolinks to it anywhere you write it.

## See also

- [Relationships](relationships.md) for the other link types.
- [The project website](website.md) for all the site sections.
- [Command reference](../reference/commands.md) for `ships` and `unlink`.
