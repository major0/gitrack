# Relationships

Tracks link to each other with five relationship verbs. gitrack stores only the
**forward** link you author and computes everything else — inverse edges,
blocking, rollups — from it.

## The relationships

| command | meaning | scheduling effect |
|---|---|---|
| `depends-on <ref> <target>` | `<ref>` waits for `<target>` | `<ref>` is blocked until `<target>` closes |
| `child-of <ref> <parent>` | `<ref>` is contained by `<parent>` | the parent waits for the child |
| `related <ref> <target>` | soft association | none |
| `duplicate-of <ref> <target>` | `<ref>` duplicates `<target>` | typically closed as a pointer |
| `ships <release> <target>` | the release shipped `<target>` | none; a record only (see [Releases](releases.md)) |

Create and remove edges:

```sh
git track depends-on session-refresh new-login-flow
git track child-of   new-login-flow  ship-auth-v2
git track related    docs-update      new-login-flow
git track duplicate-of old-bug        new-login-flow
git track ships       v1-2-0          new-login-flow
git track unlink      session-refresh depends-on new-login-flow
```

Each verb writes only the forward link (a file under the track's
`depends-on/`, `child-of/`, etc. directory). `child-of` enforces the
single-parent convention.

## Why only forward links

Inverse edges — "what blocks on me", "what are my children" — are **not stored**.
Storing both directions invites them to disagree after a merge. Instead, tools
reverse the forward links to answer inverse questions, so there is one source of
truth and nothing to desync.

## How blocking is computed

There is one link primitive (a directed edge); the graph builder only branches
on the link *type* to decide the edge's direction in the blocking graph (the
arrow points at the thing that must wait):

- `A depends-on B` → blocking edge `B → A` (A waits for B)
- `A child-of B` → blocking edge `A → B` (the parent B waits for the child A)

So `depends-on` and `child-of` block in **opposite** directions. This matters:
an open epic does **not** block its children — the children block the epic.

A track is **unblocked** when:

- every target in its `depends-on/` is closed, **and**
- every child (a track whose `child-of/` points at it) is closed.

A leaf track with no dependencies and no open children is unblocked. This is the
input to [boards and roadmaps](boards.md).

## A concrete example

An epic "Ship auth v2" has two child issues; one depends on the other:

```sh
epic=$(git track create -k epic "Ship auth v2")
login=$(git track create -k issue "New login flow")
session=$(git track create -k issue "Session refresh")

git track child-of   "$login"   "$epic"
git track child-of   "$session" "$epic"
git track depends-on "$session" "$login"
```

Now:

- **`session`** is blocked until `login` closes (its dependency).
- **`login`** is ready (no dependency, no children).
- **`epic`** is blocked until *both* children close — the parent waits for its
  children, never the other way around.

Close `login`, and `session` becomes ready; close `session` too, and the epic
becomes closeable.

## Inspecting relationships

```sh
git track show session-refresh         # prints blocked/unblocked + blockers
git track get  session-refresh.depends-on   # the forward edges (bare ids)
git track board                         # Ready vs Blocked across the project
```

## Task lists that track dependencies

A dependency-backed checklist item in a description (`- [ ] #new-login-flow
landed`) both renders its checkbox from the referenced track's state *and*
materializes a matching `depends-on` edge, so the checklist and the dependency
graph stay one source of truth. See [References](references.md) and the
[on-disk format](../FORMAT.md) §8.
