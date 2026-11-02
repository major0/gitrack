# A worked example tracker

A concrete `.gitrack/` tree illustrating the format, so the design can be read
and tested against real data. UUIDs are shortened here for readability; real
ones are full UUIDv7.

## The scenario

An epic "Ship auth v2" contains two issues. One issue depends on the other.
There is a standalone bug, marked a duplicate of one issue.

```
.gitrack/
├── config/
│   ├── kinds                     # project\nepic\nissue\nbug\ntask\nnote
│   ├── states                    # new\ntriaged\nin-progress\nin-review\ndone\nwontfix\nduplicate
│   ├── states.closed             # done\nwontfix\nduplicate
│   └── workflow/
│       └── bug                   # new -> triaged \n triaged -> in-progress ...
│
├── 0192-epic-auth/               # the epic
│   ├── kind                      # "epic"
│   ├── title                     # "Ship auth v2"
│   ├── description               # markdown body
│   ├── state                     # "in-progress"
│   └── created                   # 2026-11-02T10:00:00Z
│
├── 0192-issue-login/             # child issue 1
│   ├── kind                      # "issue"
│   ├── title                     # "New login flow"
│   ├── state                     # "done"
│   ├── child-of/
│   │   └── 0192-epic-auth        # belongs to the epic
│   └── comments/
│       └── 0192-c1.md            # a comment
│
├── 0192-issue-session/           # child issue 2
│   ├── kind                      # "task" (default; kind is cosmetic, ADR-0006)
│   ├── slug                      # "session-refresh" (typeable handle, ADR-0009)
│   ├── title                     # "Session refresh"
│   ├── state                     # "in-progress"
│   ├── description               # contains a task list (see below)
│   ├── child-of/
│   │   └── 0192-epic-auth
│   └── depends-on/
│       └── 0192-issue-login      # blocked until login is done
│
└── 0192-bug-token/               # a standalone bug
    ├── kind                      # "bug"
    ├── title                     # "Token leak in logs"
    ├── state                     # "duplicate"
    └── duplicate-of/
        └── 0192-issue-session
```

## What the dumb code computes from this (per ADR-0005)

Edge directions in the blocking graph (arrow points at the waiter):

- `issue-session depends-on issue-login` → `issue-login → issue-session`
  (session waits for login).
- `issue-login child-of epic-auth` → `issue-login → epic-auth`
  (epic waits for login).
- `issue-session child-of epic-auth` → `issue-session → epic-auth`
  (epic waits for session).

Now the queries:

- **Is `issue-session` unblocked?** Its only `depends-on` target is
  `issue-login`, whose `state` = `done` (closed ✓). It has no children. So
  `issue-session` **is unblocked** and actionable. Its parent epic being open is
  irrelevant to the child — the corrected direction (ADR-0005) gets this right.
- **Is `epic-auth` unblocked / closeable?** Its children are `issue-login`
  (done ✓) and `issue-session` (in-progress ✗). One child is open, so the epic
  is **blocked** — correct: an epic waits for its children.
- **Epic progress (rollup):** children of `epic-auth` = tracks whose `child-of/`
  contains `epic-auth` → `issue-login` (done), `issue-session` (in-progress).
  Progress = 1/2.
- **Unblocked work frontier:** `issue-session` (its dependency is done). The epic
  waits on that remaining child; the bug is closed as duplicate.

## Task lists and derived checking (ADR-0008)

`0192-issue-session/description` holds a GFM task list mixing a free item and a
dependency-backed item:

```markdown
## Plan
- [ ] write the migration guide          # free item, hand-toggled
- [ ] #new-login-flow landed              # dependency-backed (slug reference)
```

The second item references `issue-login` by its slug `new-login-flow`. Because
`issue-login/state` = `done` (closed), the **rendered** page shows that item as
`- [x]`, even though the stored bytes stay `- [ ]`. Closing `issue-login` wrote
only its own `state` file — the checklist updates purely at render time, with no
write into `issue-session`. Creating that item also materialized the
`depends-on/0192-issue-login` link shown above, so the checklist and the
dependency graph are the same data.

## Derived referenced-by (ADR-0009)

Nobody typed a UUID anywhere. `issue-login`'s `referenced-by` is computed as the
union of `git log -- .gitrack/0192-issue-login/` (every commit that touched the
track) and `git log --grep='Gitrack-Ref: new-login-flow'` (code commits that
opted in by slug). No `referenced-by/` directory is stored.

## Why this example earned its keep

A first pass (ADR-0004) treated `child-of` as blocking the child, which would
have wrongly reported `issue-session` blocked by its open epic. Materializing
this tree exposed the inverted direction before any code was written, and
ADR-0005 corrected it. The worked example is doing its job: catching modeling
errors on paper.
