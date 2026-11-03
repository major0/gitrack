# ADR-0034: Cross-repo refs resolve to an external link, not a live fetch

## Status
accepted

## Context
ADR-0011 defined the cross-repo reference syntax `#<repo-url>?track=<uuid>` but
left *resolution* open; its verification box says such a ref "parses into
(repo-url, uuid) and resolves against that remote when reachable; otherwise
renders as URL + short id." ADR-0029 (decentralized/offline/static) is the
binding constraint here: rendered output must work from `file://` with no
network for core content, and must not add a hosted/network dependency.

A live cross-repo resolution (clone or fetch the remote tracker at render time to
read the target track) would break offline rendering, add latency, and introduce
a network + trust surface — exactly what ADR-0029 rules out of the deployment
path. The target repo is also an untrusted external source.

## Decision
Cross-repo resolution is **parse + render as an external link, never a live
fetch** (first slice):

1. **Parse** (`core.ParseCrossRepo`): recognize `#<repo-url>?track=<uuid>` and
   extract `(repoURL, uuid)`. `<repo-url>` is any `http(s)://`, `ssh://`,
   `git@host:...`, or `file://` git URL; `<uuid>` may be a full or short uuid.
2. **Render** (autolinker, ADR-0030): emit a link to the remote REPO
   (`<a href="<repo-url>">#<repo-label>?track=<short-uuid></a>`), labeled with the
   repo and the short uuid. No fetch, no clone — so it stays offline, static,
   and trust-safe (the link is just a pointer; the remote is never read at render
   time). The local autolink form (ADR-0030) is unchanged and tried only when the
   token is NOT a cross-repo form.

Live resolution (reading the remote tracker to show the target's title/state) is
an explicit non-goal of this slice. If added later it must be an opt-in,
out-of-render-path step (e.g. a cache populated by an explicit `fetch`), never a
render-time network call, to preserve ADR-0029. Documented as future work.

## Consequences
- Easier: a `#<repo-url>?track=<uuid>` in a description/comment becomes a
  clickable link to the other repo — the distributed, cross-repo cross-reference
  works with zero network at render time.
- Safer/offline: no fetch, no clone, no trust surface in the render path; the
  page still opens from `file://`.
- Harder: the link shows the repo + short uuid, not the remote track's title
  (we cannot know it without fetching). Acceptable — the link is the pointer.
- Cost accepted: no verification that the remote track exists; a stale/wrong ref
  renders as a link that may 404 on the remote. Same as any hyperlink.

## Verification
- [ ] `core.ParseCrossRepo("#https://h/o/r?track=0192...")` returns the repo URL
      and uuid; a non-cross-repo token returns not-a-match.
- [ ] The autolinker renders a cross-repo ref as an `<a href="<repo-url>">` link
      labeled with the repo + short uuid, and does NOT treat it as a local ref.
- [ ] No network call happens during rendering (offline / file:// unaffected).
- [ ] A local `#<slug>`/`#<uuid>` is still resolved locally (ADR-0030 unchanged).
