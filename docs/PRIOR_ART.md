# Prior Art Review

A survey of distributed, VCS-embedded issue trackers, done to learn from their
designs and known problems before committing to our own. Most are unmaintained;
the lessons are in *why* they made the choices they did.

Content below was rephrased and summarized from public project docs for
compliance with licensing restrictions.

## The two storage philosophies

Every tool in this space picks one of two fundamentally different places to put
the tracker data. This choice determines everything else.

### 1. Separate ref namespace (data NOT in the working tree)

**git-bug** ([data model](https://github.com/git-bug/git-bug/blob/master/doc/design/data-model.md))
is the most mature and most studied example.

- Stores each entity as a chain of git commits under `refs/bugs/<id>`, outside
  the working tree. No files are added to your project checkout.
- Uses an **operation-based CRDT**: instead of a mutable snapshot, it stores an
  append-only series of `Operation`s (create, comment, set-status, ...)
  aggregated into `OperationPack` blobs. Final state is computed by replaying
  the operations in a deterministic order.
- Ordering uses **Lamport logical clocks**, not wall-clock timestamps (remote
  clocks are untrustworthy). Concurrent edits are ordered by the DAG structure,
  then by operation-pack id as a tie-break.
- Merging is conflict-free by construction: concurrent branches become a DAG
  with a single head; there is never a textual conflict to resolve.

**Strength:** beautiful, abuse-resistant, truly conflict-free merging.
**Fatal mismatch for us:** the data lives in `refs/bugs/*`, deliberately
decoupled from code commits. Merging a topic branch into main does **not** bring
the bug state with it. That is the opposite of our core requirement.

### 2. Working-tree plaintext (data IS in the checkout)

**driusan/bug**, **BugsEverywhere**, **Ditz**, **Artemis**, **GitIssius**, and
the original **gitrack** all put data in the checked-out tree.

- `driusan/bug`: one descriptive subdirectory per issue under `issues/`,
  human-readable plaintext. Its stated benefit: because issues are plaintext in
  the tree, they *branch and merge along with the rest of the code*, and
  conflicts are resolved with standard git tools. Follows the "poor man's issue
  tracker" convention (an `issues/` directory you could maintain by hand).
- **Ditz**: light-weight, VCS-agnostic (git, hg, darcs, bzr), YAML issue files.
- **GitIssius**: stored issues on a dedicated `gitissius` branch you never check
  out — a hybrid that keeps data in a tree but off to the side.
- **Artemis**: Mercurial extension, one directory per issue, Maildir-style.

**Strength:** data merges with the code as a single operation — exactly our
requirement. Human-readable and tool-independent: you can read and edit issues
with no special software.
**Weakness:** you inherit git's line-based text merge, including its documented
sharp edges (see below).

## Why we must use the working-tree model

Our hard requirement: *the tracker state is merged as part of `git-merge`*. If a
bug is fixed on a topic branch, merging that branch must bring both the code fix
and the tracker update as one operation, living together. Only the working-tree
model delivers this. git-bug's separate-ref model structurally cannot, however
elegant its merging is.

## The merge problem we inherit, and how we design around it

Putting mutable state in the tree means git's text merge applies. Known hazards
(documented in ["Analyzing and Evaluating the Behavior of Git Diff and
Merge"](https://arxiv.org/pdf/2507.22071.pdf)):

- Two sides adding different lines at the same spot may silently concatenate
  both, in arbitrary order, with no conflict reported.
- Edits to the same line on both sides produce conflict markers inside the file.

A naive "one mutable file per issue whose fields get rewritten in place" design
(early gitrack, Ditz) maximizes these collisions: every status change rewrites
the same lines.

**Our synthesis — borrow git-bug's best idea without its ref model:** make the
per-issue history an **append-only log of immutable event files**, one file per
event, each file named by a content hash so two authors never pick the same
name. Current state is *computed* by folding the events (git-bug's operation
replay), not stored as a rewritable field. Appends on different branches add
*different files*, which git merges cleanly (union of a directory) instead of
colliding on the same lines. Only genuine same-field concurrent edits need human
resolution, and those are rare and meaningful.

This gives us git-bug's conflict-resistance *and* driusan/bug's merges-with-code
property at once.

## Design lessons carried forward

1. **Store events, compute state.** Never store a mutable snapshot as the source
   of truth. (from git-bug)
2. **Immutable, content-addressed event files.** One file per event; filename is
   a hash so concurrent authors don't collide on paths. (synthesis)
3. **Don't trust wall-clock time for ordering.** Keep a logical/causal ordering;
   use timestamps only for display. (from git-bug's Lamport clocks)
4. **Human-readable, tool-independent on disk.** A person must be able to read
   and understand an issue with `cat` and no special software. (from
   driusan/bug, BugsEverywhere)
5. **One universal object.** project/epic/issue/bug/task/note are the same base
   object with a `kind` field, not separate schemas. (our requirement; gitrack
   already treated "type" as just a directory)
6. **Stable GUID identity, not a path or sequence number.** Sequence numbers
   collide across branches; paths change when an issue is reclassified. A GUID
   is branch-safe and rename-safe. (lesson from sequence-number collisions in
   driusan/bug and Ditz)
7. **The format is the product.** The CLI is replaceable; the on-disk format is
   the contract every tool and every future reimplementation depends on.
