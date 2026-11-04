# Working with tracks

Everything in gitrack is a track. This page covers creating, reading, changing,
listing, and searching them, plus the dotted-path accessors for scripting.

## Creating

```sh
git track create "New login flow"            # kind defaults to "task"
git track create -k bug "Token leak in logs" # set the kind
```

`create` prints the new track's UUID. It also:

- sets `state=open` and `status=new`;
- derives a repo-unique **slug** from the title (`new-login-flow`), which you
  can type anywhere a reference is accepted;
- commits the new track as its own revision scoped to `.gitrack/`.

The `kind` (`task`, `bug`, `issue`, `epic`, `note`, ...) is a cosmetic label; it
does not change how the track is stored or scheduled.

## Reading

```sh
git track show new-login-flow      # human-readable summary + blocked status
git track get new-login-flow       # the whole object
git track get --json new-login-flow
```

`show` prints the fields plus the computed blocked/unblocked status and any
mentions from commit history. `get` is the machine-friendly accessor (below).

## Updating

`update` is the multi-field editor and the only command that understands the
lifecycle invariants:

```sh
git track update new-login-flow -t "Login flow v2"   # title
git track update new-login-flow -k issue             # kind
git track update new-login-flow -d "Longer body..."  # description
git track update new-login-flow -s in-progress       # status (validated)
```

Flags: `-t/--title`, `-k/--kind`, `-d/--description`, `-s/--status`. The `-s`
move is validated against your status workflow (see
[Status workflows](workflow.md)); a disallowed transition is rejected. Note that
`state` is **not** an `update` flag — open/closed changes only through
`close`/`reopen`.

## Closing and reopening

```sh
git track close new-login-flow                 # state=closed, status=done
git track close dup-bug --status=duplicate     # close with a specific status
git track reopen new-login-flow                # state=open, status=reopened
```

## Listing and searching

```sh
git track list                       # everything
git track list --state open          # by state
git track list -s in-progress        # by status
git track list -k bug                # by kind
git track search "login"             # text in title/description/slug
```

## Accessor paths (`get` / `set`)

Because a field is a file and a multi-value attribute is a directory, a track is
addressable by a dotted path `<ref>.<key>[.<key>...]`. The first `.` separates
the reference (UUID/short/slug — none contain `.`) from the key path.

```sh
git track get new-login-flow.status        # a single field (scalar)
git track get new-login-flow.labels        # a multi-value attribute -> bare ids
git track get new-login-flow.comments      # bare comment UUIDs
git track get new-login-flow.comments.<id> # one comment by its bare id
git track get --json new-login-flow        # the whole object as JSON
```

`set` writes a **free-form** field only:

```sh
git track set new-login-flow.title "Login flow v2"
git track set new-login-flow.description "..."
```

`set` **refuses** the governed fields `state` and `status`, redirecting you to
`close`/`reopen` and `update -s` so the lifecycle keeps a single validated path.
Storage suffixes (like `comments/<id>.md`) are a storage detail — never part of
the id you type or the listing you read.

## Labels and assignees

These are multi-value attributes stored as directories; list them with `get`:

```sh
git track get new-login-flow.labels
git track get new-login-flow.assignees
```

## Where the data goes

Each track is a directory `.gitrack/<uuid>/` with one file per field. See the
[on-disk format](../FORMAT.md) for the full layout and the
[worked example](../EXAMPLE.md) for a concrete tree.
