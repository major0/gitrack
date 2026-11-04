# ADR-0042: A static source browser (tree, blob, log, commit) as a site section

## Status
accepted

## Context
The original third goal was "a statically rendered website for the git side of
the repository — gitweb, but with the flexibility of Hugo." Every other part of
that goal now exists; the commit/tree/blob browser (the `source` section of
ADR-0037) is the last gap.

gitweb is a CGI program that answers any (ref, path) query on demand. A static
site cannot: it must decide up front what to write. Rendering every file at
every commit is O(commits × files) pages, which is unusable for any real
history. The design question is therefore what to pre-render, and how to keep the
output bounded, safe, and offline (ADR-0029).

## Decision

### Snapshot the tree at one ref; keep history as a bounded log
- **Tree and blob pages are rendered for one ref**, default `HEAD`, resolved once
  to a commit sha. That is the browsable snapshot: every directory and every
  file at that commit.
- **History is a log of the most recent N commits** (default 100), each with its
  own commit page: message, author, date, parents, a per-file `+/-` summary, and
  the patch. A commit page shows what changed in that commit; it does not link
  into a tree at that commit, because those pages are not rendered.

This is "gitweb at one revision plus recent history" — the part of gitweb that
answers "what does the code look like, and what changed lately" — at a page count
of (directories + files + N), independent of how long history is. Browsing an
older revision means regenerating with `ref` pointed at it.

### Configuration
Two optional single-line files under `.gitrack/config/site/source/`:
- `ref` — what to snapshot (a branch, tag, or sha). Default `HEAD`.
- `commits` — how many log entries to render. Default 100.

A ref beginning with `-` is rejected so it can never be read as a git option. A
ref that does not resolve, a missing repository, or an empty history means the
section emits nothing and drops its nav entry (ADR-0026), like the other
sections.

### Git access lives in its own package
A new `internal/gitsrc` package wraps the plumbing (resolve, list tree, read
blobs, log, commit detail). It knows nothing about the tracker, so it is testable
against a throwaway repository and reusable. Notable choices:
- Blobs are read through one long-lived `git cat-file --batch` process rather
  than one process per file.
- Commit file summaries and patches come from `git diff` between the first parent
  and the commit (for a root commit, the empty tree, whose id is asked from git so
  sha256 repositories work). A merge is therefore shown against its first parent.
- `--no-ext-diff` and `--no-textconv` are always passed, so a browser run never
  executes a configured diff or textconv driver.

### Bounded output
- A blob over 512 KiB is truncated with a note; a patch over 256 KiB is
  truncated with a note. Binary files (a NUL in the first 8000 bytes, the same
  heuristic git uses) show their size instead of content.
- A commit patch is read through a limit and the git process is stopped once the
  limit is reached, so one enormous commit cannot exhaust memory.

### Page layout and addressing
Every page is a directory index so no file path can collide with a directory
path:
- `source/index.html` — the root tree.
- `source/tree/<dir>/index.html` — a directory.
- `source/blob/<path>/index.html` — a file. A repository containing both `a`
  (a file) and `a.html/` (a directory) would collide under a `<path>.html`
  scheme; under this one, a file page is always a leaf directory.
- `source/log.html` and `source/commit/<sha>.html`.

All links are relative, so the section works from `file://` and a subpath. Path
segments are URL-escaped in links. A path component that is empty, `.`, or `..`
is skipped, so a malformed tree can never write outside the output directory.

### What is shown, and what is deliberately not
- **Author name and date only; a commit's author email is never read.** The site
  is public and commit emails are personal data that the tree browser has no need
  to publish. This concerns commit metadata only: file contents and patches are
  shown verbatim, including any address a file itself contains.
- Commit messages are rendered through the existing escape-then-linkify path
  (ADR-0030), so `#refs` in a message link to their tracks, and `<script>` in a
  message is inert.
- Symbolic links show their target; submodules are listed without a link.
- **No syntax highlighting.** Blobs are escaped plain text with line numbers
  (`#L12` anchors work). Highlighting must be build-time HTML with inline CSS to
  stay offline (ADR-0038), and belongs with the planned per-language options.
- `git track serve` does not serve this section; it is part of `web` only.

## Consequences
- Easier: a project gets a browsable, linkable view of its code and recent
  history with no server, on the same offline, host-neutral footing as the rest
  of the site.
- Easier: output size is predictable and independent of history length.
- Harder: older revisions are not browsable in one site. A project that wants a
  tag-by-tag archive generates once per ref.
- Harder: a repository with very many files produces very many pages. That is
  the nature of a static tree browser; the log and patch limits bound the rest.
- Cost accepted: no highlighting and no per-path "last commit" column (that needs
  a history walk per path). Both are possible later without changing the layout.

## Verification
- [ ] With the section enabled in a git repository, `source/index.html` lists the
      root tree at the snapshot ref, with directories before files.
- [ ] Every directory has a `source/tree/<dir>/index.html` and every file a
      `source/blob/<path>/index.html`; a repository with a file `a` and a
      directory `a.html/` generates without a collision.
- [ ] A blob shows escaped content with `#L<n>` anchors; a `<script>` in a file is
      inert; a binary file shows its size; an oversized file is truncated with a
      note.
- [ ] `source/log.html` lists at most `commits` entries, newest first, and each has
      a commit page with message, author name, date, changed files, and patch.
- [ ] No commit author's email address is read or shown: the log and commit pages
      carry the author name and date only. (File contents are shown verbatim, so
      an address that is itself written in a file appears there, as it would in
      any source viewer.)
- [ ] A `#ref` in a commit message links to its track page.
- [ ] `ref` selects a tag or branch; a ref starting with `-` is rejected; an
      unresolvable ref, a non-git directory, or an empty history emits nothing and
      leaves no nav entry.
- [ ] A patch over the limit is truncated with a note, and the git process stops.
- [ ] All hrefs are relative and URL-escaped, so paths containing spaces, `#`, or
      `%` work.
