# ADR-0031: Commit "mentions" of a track are derived from git history

## Status
accepted

## Context
The stakeholder wants GitHub-style back-references: when a code commit mentions a
track — `closes #<id>`, `resolves #<id>`, `relates to #<id>`, `related to #<id>`,
or a bare `#<id>` — the track should show a `mentions` section listing those
commits, so you can cross-reference the source tree against the tracker.

This is the explicit half of ADR-0009's `referenced-by` concept, which already
established that references-from-commits are **derived from git history, not
stored** (storing a `mentions/` directory would duplicate git and re-introduce
the write-on-write hazard — a code commit would have to mutate a track). ADR-0009
keyed the explicit signal on a `Gitrack-Ref:` trailer; the GitHub-style keywords
are a more natural signal and supersede the trailer as the primary form.

Portability note: this host's git (a fork) is not built with PCRE, so
`git grep -P` fails. The derivation must avoid PCRE — a fixed-string
`git log --grep` candidate filter plus matching in Go is portable
(verified in `tmp/probe-mentions.sh`).

## Decision
A track's **`mentions`** is a derived view (never stored): the commits whose
message references the track via `#<handle>`, classified by keyword.

Derivation (`core.Mentions(id)`):
1. For each of the track's handles (its `slug`, slug aliases, and a short uuid),
   get candidate commits with `git log --grep="#<handle>" -F` (fixed string, no
   PCRE — portable).
2. For each candidate, read the full message (`git log -1 --format=%B`), confirm a
   `#<handle>` token is present, and classify the mention verb:
   - `closes` / `resolves` / `fixes` (+ `#<handle>`) -> **closes**
   - `relates to` / `related to` (+ `#<handle>`) -> **relates**
   - otherwise -> **mentions**
3. Return commit sha + subject + verb, de-duplicated by sha, newest first.

gitrack does **not** auto-close a track on a `closes #<id>` commit. GitHub
auto-closes on merge-to-default, which is a hosted-service behavior; gitrack has
no server and closing is an explicit `git track close` (ADR-0028). The `closes`
verb is captured for display only.

Surfacing: `git track show` lists mentions; the web generator renders a Mentions
section on the track page (commit short-sha + subject + verb). The structural
half of ADR-0009 (`git log -- .gitrack/<id>/`) remains available as the
"activity" complement; `mentions` is specifically the message-keyword view.

## Consequences
- Easier: natural GitHub-style cross-referencing — write `closes #login-flow` in
  a commit and the track shows it, with no tracker write and no ceremony.
- Easier: nothing to keep consistent; mentions are always accurate against
  history and never drift (derive-don't-duplicate, ADR-0009/0026).
- Portable: no PCRE dependency; works on the git-fc fork and stock git alike.
- Harder: computing mentions is a `git log` scan per track handle; O(commits)
  per lookup, cacheable later. Acceptable at show/generate time.
- Cost accepted: no auto-close (deliberate — no server; closing is explicit).
  Cross-repo mention sources are out of scope here (local history only).

## Verification
- [ ] A commit message `closes #<slug>` makes that commit appear in the track's
      mentions with verb `closes`; `relates to #<slug>` -> `relates`; a bare
      `#<slug>` -> `mentions`.
- [ ] An unrelated commit (no `#<handle>`) is not listed.
- [ ] Derivation uses no PCRE (`git grep -P`); it works on a git build without
      libpcre.
- [ ] A `closes #<id>` commit does NOT change the track's state/status.
- [ ] `git track show` and the web track page list the mentions.
