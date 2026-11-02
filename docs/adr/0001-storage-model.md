# ADR-0001: Tracker state lives in the working tree as an append-only event log

## Status
superseded by ADR-0003

## Context
The product's defining requirement: the tracker state must merge as part of an
ordinary `git merge`. When a bug is worked on a topic branch, merging that
branch into the mainline must bring the code change *and* the tracker update
together, as a single operation, living in the same commit history.

Prior art (see `docs/PRIOR_ART.md`) splits into two camps:

- **git-bug** stores entities in a separate ref namespace (`refs/bugs/*`) as an
  operation-based CRDT. Merging is conflict-free, but the data is deliberately
  decoupled from code commits — a topic-branch merge does *not* carry it. This
  structurally cannot satisfy our requirement.
- **driusan/bug, BugsEverywhere, Ditz, gitrack** store data in the working tree,
  so it branches and merges with the code. This satisfies the requirement but
  inherits git's line-based text merge, whose documented sharp edges punish any
  design built on rewriting the same lines in a mutable file.

We need the working-tree property without the rewrite-collision problem.

## Decision
We will store all tracker data in the working tree under a top-level `.gitrack/`
directory, and we will model each object's history as an **append-only log of
immutable, content-addressed event files** (one file per event, filename derived
from a hash of the event content). Current state is **computed** by folding the
events in causal order, never stored as a rewritable field. We adopt git-bug's
"store operations, compute state" insight while keeping the data in the tree so
it merges with the code.

## Consequences
- Easier: a topic-branch merge brings tracker changes with the code, as required.
- Easier: concurrent edits on different branches append *different* files, which
  git merges as a clean directory union — no textual conflict on the common case.
- Easier: full offline use; issues are readable with `cat` and no special tool.
- Harder: computing current state requires replaying events (a fold), not a
  single file read. We accept this cost; the event count per object is small.
- Harder: genuinely concurrent edits to the *same* field still require human
  resolution, but these are rare and semantically meaningful.
- Cost accepted: we do not get git-bug's fully automatic CRDT merge. We trade a
  small amount of conflict-freeness for the merges-with-code property, which is
  the whole point of the product.

## Verification
- [ ] A fresh repo gains a `.gitrack/` directory tracked in the working tree.
- [ ] Creating an object writes a new file per event; no event file is ever
      modified in place after creation (verified by hashing filenames == content).
- [ ] On two branches, adding a comment to the same object on each, then merging,
      produces both comment files with no conflict.
- [ ] Reading an object's state replays its event files rather than reading a
      stored snapshot (no mutable `status`/`state` file is the source of truth).
