# Comments

Comments are threaded Markdown discussion on a track. gitrack stores only the
comment body — the author and date come from git — so comments merge cleanly and
carry no redundant metadata.

## Adding a comment

```sh
git track comment new-login-flow -m "Spike looks good; starting the form."
```

With no `-m`, your `$EDITOR` opens on a temporary file for a longer body. The
body is stored verbatim as Markdown.

## Replying (threading)

```sh
git track comment new-login-flow --reply-to <comment-id>
```

`--reply-to` sets the parent comment and, in editor mode, seeds the buffer with
the parent cite-quoted as a Markdown blockquote (git-commit style). The thread
tree is reconstructed from each comment's `reply-to`; a comment with none is
top-level, and an orphaned reply (whose parent is missing) falls back to
top-level.

## Author and date come from git

A comment file stores **no** author and **no** date. git already records both on
the commit that added the comment, so gitrack derives them:

- author = the commit author of the file's first appearance
- date = the author-date of that commit

The only metadata a comment may carry is `reply-to` (the parent's id), which git
cannot derive. This is why comments need no front matter unless they are a reply.

## How comments are stored

Each comment is a file `comments/<uuid>.md` inside the track's directory, named
by a UUIDv7 so comments minted offline on different branches never collide and
the id stays stable across edits. Because the directory is append-only,
concurrent comments from different people merge as a clean union — no conflict.

List a track's comments by their bare ids, or read one:

```sh
git track get new-login-flow.comments        # bare comment UUIDs
git track get new-login-flow.comments.<id>   # one comment body
```

## How comments render

The website renders a track's comments as a **threaded tree** on its page:
replies nest under their parent, each comment shows its git-derived author and
date, and `#refs` in comment bodies are autolinked
([References](references.md)). The same threading appears in the live preview
(`git track serve`).

## Editing and deleting

A comment is just a file. Edit it with your editor (the id is stable, so
`reply-to` references survive), or remove the file to delete it. As with all
tracker changes, commit through gitrack or stage and commit yourself; the
author/date shown will reflect the commit history.
