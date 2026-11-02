# ADR-0033: `git track commit` applies the message's own ref directives

## Status
accepted

## Context
ADR-0032 added `git track commit --resolve <id>` to close a track in the same
revision. The stakeholder wants the richer, GitHub-like model: the **commit
message itself** carries the directives. gitrack should process the message —
find `closes #<id>` / `resolves` / `fixes` and `relates to #<id>` — apply the
corresponding track updates, amend the track files into the commit, and commit.
It should also accept a pathspec (`-- <file>...`) and let the user name a track
positionally and set the close status directly.

So the message is the interface, not just flags. `--resolve`/`<id>` become sugar
that inject a `closes #<id>` directive; writing `closes #<id>` in the message
does the same thing. This supersedes ADR-0032's flag-centric model while keeping
its outcome and its safety stance (ADR-0031: only an explicit gitrack action,
never passive observation, changes state).

## Decision
`git track commit [<id>...] [--status <s>] [-m <msg>] [-- <file>...]`:

1. **Obtain the message.** From `-m`, or by opening `$EDITOR` (like `git commit`)
   when `-m` is absent — so gitrack can read the message file the user wrote.
2. **Parse directives from the message** (`core.ParseDirectives`): every
   `<verb> #<handle>` occurrence, verb-classified as ADR-0031 does
   (closes/resolves/fixes -> close; relates/related to -> relate; bare -> mention).
   Positional `<id>` args and `--resolve` are sugar that prepend a `closes`
   directive for those ids.
3. **Apply the directives:**
   - a **close** directive closes that track (status `done`, or `--status <s>`).
   - a **relate** directive adds a `related` relationship FROM each other close/
     relate target where meaningful; minimally it is recorded by the message and
     surfaced as a mention (ADR-0031) — no state change. (First slice: relate/
     mention directives apply no mutation; only closes mutate. Relationship
     creation from a relate directive is a documented follow-on.)
   - unresolvable `#<handle>` tokens are left alone (not every `#` is a ref).
4. **Amend the track files into the commit and commit.** Stage the tracker
   changes (`.gitrack/`), then commit: the staged index when no pathspec is
   given, or exactly the given `-- <file>...` plus the tracker changes when a
   pathspec is given. One revision contains the code + the applied track updates,
   and the message (which already names the directives) is the record.

Still: gitrack changes state only through an explicit action (`close`, or this
`commit`); passively observing a `closes` commit never auto-closes (ADR-0031).

## Consequences
- Easier: the commit message is the single, familiar interface — write `closes
  #login` and committing through gitrack closes it, bundled with the code.
- Easier: `-- <file>...` scopes the commit to named files (like git); no pathspec
  commits the staged index (ADR-0032 behavior retained).
- Easier: `--status` and positional `<id>` are ergonomic shortcuts over the
  message directives.
- Harder: gitrack parses commit messages; the parser must be conservative
  (resolve-gated handles, verb word-boundaries) to avoid acting on a stray `#`.
  Reuses the ADR-0031 classifier.
- Cost accepted: relate-directive -> relationship creation is deferred; relate/
  mention directives are informational (surfaced via mentions) in this slice.

## Verification
- [ ] `git track commit -m "fix thing\n\ncloses #<slug>" -- app.go` closes the
      track and commits app.go + the tracker change as one revision.
- [ ] With no `-m`, `$EDITOR` opens; the written message's directives are applied.
- [ ] `--status <s>` sets the close status for closed tracks; positional `<id>`
      acts as a `closes` directive.
- [ ] A bare/unresolvable `#token` in the message changes nothing.
- [ ] No pathspec commits the staged index; `-- <files>` scopes to those files
      plus the tracker change.
- [ ] Passive `closes #<id>` commits (outside this command) still do not
      auto-close (ADR-0031).
