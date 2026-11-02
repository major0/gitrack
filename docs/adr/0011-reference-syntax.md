# ADR-0011: Reference syntax — `#<uuid>`, `#<short>`, slug, and cross-repo

## Status
accepted

## Context
References need a concrete syntax. Prior ADRs used a `gitrack:<handle>` sigil and
a typeable slug (ADR-0009), because a raw UUID is not typeable. The stakeholder
now wants:

- `#<uuid>` and a short form `#<short-uuid>` (git-short-hash style), and
- a cross-repo form `#<repo-url>?track=<uuid>` to reference tracks in *other*
  repositories (the "distributed" in distributed tracker).

The `#` sigil matches GitHub's issue-reference idiom and is more natural than
`gitrack:`. Short-uuid resolution mirrors git's shortest-unique-prefix rule,
which we already adopted.

## Decision
A reference is written with a leading `#` and resolves to a track UUID. Forms:

| form                              | resolves to | notes |
|-----------------------------------|-------------|-------|
| `#<uuid>`                          | that UUID   | full canonical id |
| `#<short>`                         | a UUID      | shortest unique hex prefix, resolved like a git short hash; ambiguity is an error listing candidates |
| `#<slug>`                          | a UUID      | the track's authored `slug` or an alias (ADR-0009) |
| `#<repo-url>?track=<uuid>`         | a remote UUID | cross-repo; `<repo-url>` is a clonable git URL, `<uuid>` may be short |

`#<slug>` and `#<short>` are disambiguated by shape: a value matching the slug
charset (kebab-case, contains a non-hex char or a hyphen group that is not a UUID
field) is tried as a slug; a pure-hex value is tried as a short-uuid; a full UUID
is exact. The CLI resolves and errors on ambiguity.

The `Gitrack-Ref:` commit trailer (ADR-0009) accepts the same forms after the
`#` is dropped (a trailer value `new-login-flow` or `0192f1a2`), since trailers
are already keyed syntactically.

This supersedes the `gitrack:<handle>` sigil from ADR-0008/0009 wherever it
appears in running text and task-list items: the sigil is now `#`. The
`Gitrack-Ref:` trailer name is unchanged.

## Consequences
- Easier: references read like GitHub (`#...`), and short-uuids keep UUIDs
  typeable in practice.
- Easier: cross-repo refs make a track in repo A able to depend on a track in
  repo B — real distributed tracking.
- Harder: a cross-repo ref cannot be resolved offline without the remote; the
  renderer must degrade to showing the URL + short id when the remote is absent,
  and `fsck` (ADR-0012) validates/refreshes remote refs when reachable.
- Harder: slug-vs-short-uuid disambiguation needs a clear rule (above); a slug
  that looks like hex is rejected at mint time to keep resolution unambiguous.
- Cost accepted: cross-repo resolution introduces network and trust surface;
  remote content is untrusted and only its id/state is read, never executed.

## Verification
- [ ] `#<uuid>`, `#<short>`, and `#<slug>` all resolve to the correct local UUID,
      with ambiguous `#<short>` erroring and listing candidates.
- [ ] A slug that is pure hex (ambiguous with a short-uuid) is refused at mint.
- [ ] `#<repo-url>?track=<uuid>` parses into (repo-url, uuid) and resolves against
      that remote when reachable; otherwise renders as URL + short id.
- [ ] Task-list items and running-text refs use `#`, not `gitrack:`.
