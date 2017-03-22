# README

Gitrack is a distributed issue/bug/note/whatever tracker for git (and possibly
others) written in POSIX shell and inspired by
[BugsEverywhere](http://www.bugseverywhere.org). Gitracker is
different from BugsEverywhere in that it tries to be a generic interface for
tracking .. well .. anything (not just bugs) and aims to support file
attachments, threaded comment history, and customizable workflow.  All while
being written in POSIX shell script.

## Benefits

 - Bugs/Notes/Issues live with their respective code.  When you merge a bugfix
   branch that uses Gitrack, you merge the tracker history along with the coded
   bugfix.

 - Full off-line access to your bugs, notes, issues as well as being able to
   comment on other users notes in a forum-style development process.

 - Automatic generation of a markdown web view of the tracker database during
   `git push` so that all entries are browsable from GitHub/GitLab (modifying
   issues from the webinterface would require specialized software on the
   server-side).

## Warning

This code is very young (less than 20 commits at the time of this writing).
The work is still in flux and many features are not yet complete and the
tracker database format may or may not change.

## Installation

Simply copy `git-track` to a directory in your `PATH`.  From there you can use
the tracker as a git sub-command.

Example:

```
$ git track help commands
git-track commands:
  config               Modify the tracker config
  list                 List existing tracks and track types
  new                  Create a new track
  update               Update/Modify an existing track
  show                 Display the specified track
  close                Close an existing track (wrapper to update)
  reopen               Reopen a closed track (wrapper to update)
  comment              Comment on an existing track

Report bugs to http://github.com/major0/gitrack
```

## Getting Started

To create your first track simply pick a `topic`, such as bug, issue, note,
whatever you want to open the track under:

```
$ git track new headache -m 'An example track'
[master a800eaf] New headache(1d59d42): An example track
 4 files changed, 11 insertions(+)
 create mode 100644 .gitrack/headache/1d59d4242a2bd039ee39365d718c946768e00a0b/head
 create mode 100644 .gitrack/headache/1d59d4242a2bd039ee39365d718c946768e00a0b/refs/1d59d4242a2bd039ee39365d718c946768e00a0b
 create mode 100644 .gitrack/headache/1d59d4242a2bd039ee39365d718c946768e00a0b/state
 create mode 100644 .gitrack/headache/1d59d4242a2bd039ee39365d718c946768e00a0b/status
$
```

To list existing types of tracks invoke the `list` command:
```
$ git track list
Types: headache
$
```

Or specify a type to list tracked entries:
```
$ git track list headache
headache/1d59d42:   open:new        unassigned  An example track
$
```

Checkout `git track help <command>` for help on a specific command.

## License

Like `git`, this project is licensed under the GPLv2 (not GPLv2.2, GPLv3, etc).
For more information see the COPYING file.

## Author

Mark Ferrell
