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
- **Files.** Each file has a page with numbered lines. Click a line number, or
  add `#L12` to the address, to link straight to a line.
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

A `commits` value that is not a positive number is ignored and the default is
used. A `ref` that does not resolve to a commit means there is nothing to browse,
so the section produces nothing.

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

- **No syntax highlighting.** Files are shown as plain escaped text. Content is
  always escaped, so a `<script>` in a source file is displayed, never run.
- **No live preview.** `git track serve` does not serve this section; it is part
  of `git track web` only.
- **No per-file history.** There is no "last commit touching this file" column.

## See also

- [The project website](website.md) for all the site sections.
- [Configuration](configuration.md) for every config file.
- [Publishing](publishing.md) to put the generated site on a host.
