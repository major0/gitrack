# Browsing source

The `source` section turns your repository into a browsable website in the
style of gitweb: the file tree, each file with line numbers, and the recent
commit history. It is static HTML, so it works from `file://` or any dumb host
with no server running.

## Turn it on

Add `source` to `.gitrack/config/site/sections`:

```
issues
docs
source
```

Then generate the site as usual:

```sh
git track web --out ./site
```

You get a `source` entry in the top navigation. If the directory is not a git
repository, or has no commits yet, the section produces nothing and adds no
navigation entry.

## What you get

- **The tree.** `source/index.html` lists the top-level directory, directories
  first, then files with their sizes. Every directory has its own page, with
  breadcrumbs back up the path.
- **Files.** Each file has a page with numbered lines, syntax-highlighted when
  its language is recognized. Click a line number, or add `#L12` to the address,
  to link straight to a line.
- **The log.** `source/log.html` lists recent commits, newest first, with date
  and author.
- **Commits.** Each logged commit has a page with its message, author, date, its
  parents, a summary of changed files with added and removed line counts, and the
  patch with added and removed lines colored. A merge commit is shown against its
  first parent.

A `#ref` written in a commit message links to its track, the same as anywhere
else on the site. So `Closes #login-flow` in a commit message becomes a link to
that issue's page.

The link works the other way too. A track's page lists the commits that mention
it, and with the source section enabled each of those commits links to its commit
page. A commit that has no page stays plain text: that means one older than the
log (see below), or one that is not in the history of the ref you snapshot.

Binary files show their size instead of content. A symbolic link shows its
target. A submodule is listed but has no page.

## One snapshot, plus recent history

A static site cannot answer "show me this file at that commit" for every file at
every commit, so the section renders one **snapshot** of the tree and a bounded
**log**:

- The file tree is the repository at a single ref, which is `HEAD` by default.
- The log is the most recent commits, 100 by default. Each has a commit page
  showing what that commit changed. A commit page does not link into the tree as
  it was then.

To browse an older release, generate once with the ref pointed at it. A common
pattern is one site per tag.

## Configuration

Two optional files under `.gitrack/config/site/source/`, each a single line:

| file | meaning | default |
|---|---|---|
| `ref` | the branch, tag, or commit to snapshot | `HEAD` |
| `commits` | how many log entries to render | `100` |

```sh
mkdir -p .gitrack/config/site/source
echo v1.2.0 > .gitrack/config/site/source/ref        # snapshot a tag
echo 250    > .gitrack/config/site/source/commits    # a longer log
```

Highlighting has its own site-wide setting, described below.

A `commits` value that is not a positive number is ignored and the default is
used. A `ref` that does not resolve to a commit means there is nothing to browse,
so the section produces nothing.

## Highlighting

Files are colored at generation time, so the pages contain no script and work
offline like everything else. The language comes from the file name: its
extension, or a well-known name such as `Makefile` or `Dockerfile`. Gitrack does
not guess a language from a file's contents, so a file whose name it does not
recognize, and plain `.txt` files, are shown uncolored.

One setting, `.gitrack/config/site/highlight`, covers the whole site: these source
pages and also fenced code blocks in docs, blog, portal, and release-notes pages
(see [The project website](website.md)). It picks the color style. The default is
`github`, a light style. Other choices include `github-dark`, `monokai`,
`dracula`, `nord`, `solarized-light`, `solarized-dark`, `vs`, and `xcode`. A name
that is not a style is ignored and the default is used. Set it to `off` to turn
highlighting off everywhere.

```sh
echo monokai > .gitrack/config/site/highlight   # a different color style
echo off     > .gitrack/config/site/highlight   # no highlighting at all
```

Highlighting is the slowest part of generating the source section: expect about
a second for every 300 KB of source. To keep that bounded, three limits apply,
and a file that hits one is shown as plain numbered text with a note saying why:

- a file larger than 128 KiB is not highlighted;
- a file that takes more than a couple of seconds to color is abandoned;
- one generation highlights at most 32 MiB in total, counting these source files
  and the code blocks in your Markdown pages together. Everything is taken in the
  same order every time, so the same repository always highlights the same
  files, whatever the machine.

For a very large repository, `off` is the way to keep generation fast. Turning
highlighting on also makes the generated site larger, since every colored token
is marked up. The `git-track` binary itself is about 4 MB larger for carrying the
language definitions, whether or not you use the feature.

## Limits

To keep the output bounded, a file larger than 512 KiB is cut off with a note
saying how much is shown, and a commit's patch longer than 256 KiB is cut off the
same way. The summary of changed files is always complete.

The number of pages is the number of directories, plus files, plus logged
commits. That does not depend on how long your history is, but a repository with
many thousands of files will produce many thousands of pages.

## Privacy

Author **names** and dates are shown. Author **email addresses** are never read
from commits, so they cannot leak onto a public site. File contents and patches
are shown exactly as stored, so an address that is written inside a file does
appear there, as it would in any source viewer.

## What it does not do

- **No highlighting in patches.** A commit's patch is colored by added and removed
  lines only, not by language, since a patch mixes languages and shows partial
  lines. Content is always escaped, so a `<script>` in a source file or a patch is
  displayed, never run.
- **No live preview.** `git track serve` does not serve this section; it is part
  of `git track web` only.
- **No per-file history.** There is no "last commit touching this file" column.

## See also

- [The project website](website.md) for all the site sections.
- [Configuration](configuration.md) for every config file.
- [Publishing](publishing.md) to put the generated site on a host.
