# ADR-0032: `git track commit --resolve` wraps git commit and closes the track

## Status
accepted

## Context
ADR-0031 deliberately does NOT auto-close a track when a commit message says
`closes #<id>`: gitrack has no server to watch commits/merges, so passively
mutating a track from an observed commit would be a surprising write-on-write.

The stakeholder wants the convenient outcome (a resolving commit closes the
track) without that hazard, via an explicit tool action: `git track commit
--resolve <id>` wraps `git commit` of the currently staged files AND closes the
named track, recording it as one revision. This is not passive observation — the
user is driving both the commit and the close through gitrack in a single
deliberate step — so it is safe and unsurprising, and it lands the code fix and
the tracker close together (the merges-with-code property, done on purpose).

This refines ADR-0031 (passive mentions stay display-only) and complements
ADR-0028 (per-command commits scoped to .gitrack/): the commit WRAPPER has a
deliberately broader scope — the user's whole staged index plus the tracker
change — because its purpose is "commit my code work and resolve the track."

## Decision
Add `git track commit`:

- `git track commit -m <msg>` commits the **currently staged files** (the whole
  index) via the system git — a thin wrapper over `git commit`, using the user's
  identity/config/hooks.
- `--resolve <id>` (repeatable) additionally **closes** each named track
  (state=closed, status=done unless overridden) and stages the tracker change
  into the SAME commit, and appends a `closes #<slug>` line to the message so the
  mention (ADR-0031) is recorded in history. The result is one revision
  containing the staged code + the track close + the closing reference.
- `--status <s>` sets the closing status for the resolved tracks (default
  `done`).
- Without `--resolve`, `git track commit` is just a staged-index commit wrapper
  (no tracker change) — handy for committing a batch of `--no-commit` tracker
  edits plus code in one revision.

The commit scope here is the full staged index by design (contrast ADR-0028,
where each tracker mutation scopes to .gitrack/): the point of the wrapper is to
bundle the user's staged work with the resolve. If the user wants only the
tracker change committed, they use the individual commands.

gitrack still never closes a track from merely *observing* a commit (ADR-0031);
closing happens only through `git track close` or this explicit
`commit --resolve`.

## Consequences
- Easier: one command finishes a bug — commit the fix and close the track in a
  single revision whose message reads `closes #<slug>`, so the mention view
  (ADR-0031) and the closed state agree and ship together.
- Easier: the "close in the same commit as the fix" workflow is explicit and
  ergonomic, not a passive surprise.
- Harder: this command commits the whole staged index (broader than ADR-0028's
  scoped commits); documented clearly so users know it is the code+tracker
  wrapper, not a tracker-only command. A pre-commit hook failure surfaces as the
  git error.
- Cost accepted: no merge-time automation (no server); resolve is at commit time
  by the author's choice.

## Verification
- [ ] `git track commit -m msg --resolve <id>` creates ONE commit containing the
      user's staged files AND the track's state=closed change, with `closes
      #<slug>` in the message.
- [ ] After it, the track's state is `closed` (status `done` or `--status`).
- [ ] That commit appears in the track's mentions (ADR-0031) with verb `closes`.
- [ ] `git track commit -m msg` with no --resolve commits the staged index and
      changes no track.
- [ ] Passive `closes #<id>` commits (not via this command) still do NOT
      auto-close (ADR-0031 unchanged).
