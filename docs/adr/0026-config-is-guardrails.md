# ADR-0026: Config is guardrails, not enforcement; `fsck` detects divergence

## Status
accepted

## Context
The on-disk format is an open directory tree, readable and writable with no
special software (ADR-0003 made this a goal). A user can `printf 'in-review' >
.gitrack/<id>/status`, `mkdir` a relationship, or create a track in a `kind` not
listed in `config/kinds` — bypassing the `git track` CLI entirely. A git merge
can also produce field values no CLI path would have written.

Therefore the workflow/status/kind configuration cannot be an *enforcement*
mechanism. The CLI validates only the write path it controls (`update --status`
checks transitions; `config status add` checks targets). The authoritative truth
is whatever the tree actually contains.

## Decision
We state as a design tenet: **`config/` is guardrails, not enforcement.** Config
files (`config/statuses/`, `config/kinds`, ...) describe *intended* structure and
make the CLI's well-behaved path consistent; they do not and cannot constrain
what the tree may contain, because the tree is openly editable and merges
outside the CLI.

The consequence is that **`fsck` is the reconciler between intent and reality.**
Beyond rebuilding the derived cache (ADR-0014), `fsck` reports where authoritative
data diverges from the declared guardrails, as **warnings** (not errors, and
never auto-fixes of authored data — ADR-0007/0009 forbid mutating authored
fields):

- a track whose `status` is not a defined status (built-in or `config/statuses/`),
- a track whose `kind` is not in `config/kinds` (when that allow-list exists),
- a track whose `state` is not `open`/`closed`,
- (already) dangling relationship links and stale cache entries.

Warnings are advisory: the data is still valid and usable; fsck surfaces the
divergence so a human can decide (add the missing status to config, or correct
the track). `fsck` exits non-zero when warnings or cache drift exist, so CI can
gate, but it never rewrites a track's authored fields to match config.

## Consequences
- Easier: the mental model is honest — the CLI helps, the tree rules, fsck
  reconciles. No false sense that config "prevents" anything.
- Easier: hand-edits and merges are first-class; you can always drop to files,
  and fsck tells you where you drifted from intent.
- Harder: consumers must tolerate values outside config (an unknown status is
  displayed, not rejected on read). Already true for the read path.
- Cost accepted: fsck does more work (cross-checking every track against config).
  O(tracks), local, and only on demand.

## Verification
- [ ] Docs state that config is guardrails, not enforcement (FORMAT.md).
- [ ] A hand-created track with an undefined `status` is readable/usable; `fsck`
      reports it as a warning and exits non-zero.
- [ ] A track with a `kind` not in `config/kinds` (when present) is flagged.
- [ ] A track with a `state` other than open/closed is flagged.
- [ ] `fsck` never rewrites a track's authored fields to match config.
