# gitrack documentation

gitrack is a distributed issue, task, and note tracker whose state lives in your
git working tree and merges with your code. There is no server to run: clone the
repo and you have the whole tracker — issues, comments, relationships, boards,
and history — offline and complete.

This documentation is for people **using** gitrack on a project. It is organized
by subject:

## Guide

- [Getting started](guide/getting-started.md) — install, initialize, create your
  first track, see it on a board and a website.
- [Concepts](guide/concepts.md) — the one-object model, the two lifecycle axes,
  and why the data lives in the working tree.
- [Working with tracks](guide/tracks.md) — create, read, update, list, search,
  and the dotted-path accessors.
- [Status workflows](guide/workflow.md) — the fixed `state` axis, the
  customizable `status` workflow, and how to shape it with `config`.
- [Relationships](guide/relationships.md) — depends-on, child-of, related,
  duplicate-of, ships, and how blocking is computed.
- [Boards and roadmaps](guide/boards.md) — the Ready/Blocked view and the
  dependency digraph.
- [References and autolinking](guide/references.md) — the `#ref` syntax, slugs,
  short ids, cross-repo links, and how they render.
- [Comments](guide/comments.md) — threaded Markdown discussion, authored by git.
- [Commit integration](guide/commit-integration.md) — closing and relating
  tracks from commit messages, and keeping a fix and its close in one commit.
- [Releases](guide/releases.md) — record what each release shipped, and publish
  dated release notes.
- [Browsing source](guide/source.md) — a gitweb-style browser of your files and
  recent commits, as static pages.
- [The project website](guide/website.md) — generate a static site: issues,
  portal, blog, docs, and releases sections, plus the live preview server.
- [Configuration](guide/configuration.md) — every `.gitrack/config/` knob:
  kinds, statuses, and the site config.
- [Publishing](guide/publishing.md) — deploy the static site to Pages, S3, a
  shared drive, or IPFS.

## Reference

- [Command reference](reference/commands.md) — every `git track` subcommand,
  its flags, and what it writes.
