# Getting started

This walks you from an empty repository to a tracked project with a board and a
browsable website — in a few minutes, with no server anywhere.

## Install

gitrack is a single Go binary named `git-track`. Build and install it onto your
`PATH`:

```sh
go install github.com/major0/gitrack/cmd/git-track@latest
```

Because the binary is named `git-track` and lives on your `PATH`, git runs it as
a subcommand: `git track <command>` and `git-track <command>` are the same. You
can confirm the wiring with:

```sh
git track --help
```

## Initialize a tracker

From inside a git repository:

```sh
git track init
```

This creates the `.gitrack/` directory that holds the tracker and commits the
scaffold as its own revision using your git identity. (Pass `--no-commit` to
leave it staged instead.) The tracker now travels with your repository: anyone
who clones has the complete tracker, offline.

## Create your first track

A **track** is the one universal object — an issue, a task, a bug, a note, an
epic. Create one:

```sh
git track create "Build the login flow"
```

gitrack prints the new track's id (a UUID). It also derived a typeable **slug**
from the title (`build-the-login-flow`) that you can use instead of the UUID.
The track starts with `state=open` and `status=new`.

Add a bug and mark it blocked by the first track:

```sh
git track create -k bug "Session token leaks into logs"
git track depends-on session-token-leaks-into-logs build-the-login-flow
```

## See what you have

```sh
git track list                 # all tracks
git track show build-the-login-flow
git track board                # Ready vs Blocked work
```

The board shows `build-the-login-flow` as **Ready** and the bug as **Blocked**
(it waits on the login flow), because `depends-on` means "blocked until the
target closes."

## Move work along

```sh
git track update build-the-login-flow -s in-progress    # move the status
git track comment build-the-login-flow -m "Spike looks good, starting the form."
git track close build-the-login-flow                     # state=closed, status=done
```

Closing the login track makes the bug **Ready** on the next board render — no
cascading writes, just a recomputation from the authored data.

## Browse it as a website

Generate a self-contained static site — no server, works from `file://`:

```sh
git track web --out ./site
```

Open `./site/index.html` in a browser. You get a tracks index, an issues view,
per-track pages with threaded comments, and a board with a dependency graph.
To preview the *live* tracker instead (re-reads the tree on every request):

```sh
git track serve
```

It binds a random loopback port and prints the URL.

## Where to go next

- [Concepts](concepts.md) explains the model you just used.
- [Status workflows](workflow.md) shows how to customize the lifecycle.
- [The project website](website.md) turns the site into a full portal with docs
  and a blog.
