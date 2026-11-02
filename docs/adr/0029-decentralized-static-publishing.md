# ADR-0029: Rendered output is static and deployable to decentralized transports

## Status
accepted

## Context
A stated end-goal (see `docs/VISION.md`) is to run the tracker with **no hosted
service** — syncing via decentralized transports (shared drives, S3 buckets,
drop-boxes, IPFS) and serving views as static content. The motivations are
concrete: no server to own or trust, a far smaller attack surface, lower energy
use, and offline resilience.

Much of this is already true: the format is git-portable (ADR-0003), and the
planned generators emit static HTML (ADR-0010). What has not been written down
is the *constraint* this goal places on every current and future view, and the
intent to treat non-HTTP transports as first-class. Without that, a future
feature could quietly add a runtime-server dependency and break the goal.

## Decision
We commit the architecture to decentralized, static-first delivery:

1. **No server of record.** The on-disk format under `.gitrack/` is the only
   source of truth. gitrack never requires a hosted application to be correct,
   usable, or collaborative. Collaboration is git sync over any remote.
2. **Every rendered view is static.** The web view, issues view, boards, and
   roadmaps render to self-contained static HTML + assets that work from a dumb
   host or opened directly as `file://`. No view may require a running server in
   the deployment path. `git track serve` (ADR-0024) is a LOCAL PREVIEW only and
   is never the deployment artifact.
3. **Transport-agnostic publishing.** The generated site is a plain directory, so
   it deploys by whatever copies files: `aws s3 sync`, a copy to a shared drive,
   an IPFS add, rsync, or the GitHub Pages workflow (ADR-0010). gitrack provides
   a generator whose output is a directory; it does not couple to one host. A
   thin `publish`/deploy helper for common targets (S3, filesystem) may be added
   later, but the generator output remains host-neutral so no helper is required.
4. **Offline-complete.** Static output must render without network calls for its
   core content. (A CDN script such as Mermaid is a progressive enhancement, not
   a hard dependency; a fallback — pre-rendered or a notice — keeps the page
   usable offline. The `web` generator should vendor or inline such assets where
   practical so a `file://` open works with no network.)

This binds the forthcoming `web`/`issues` generators: their output is a static
directory, self-contained, deployable to any of the above, and openable offline.

## Consequences
- Easier: the tracker runs anywhere git goes and publishes anywhere files go;
  no hosted dependency, minimal attack surface, low energy, offline-first.
- Easier: publishing is "copy a directory," so new transports need no new code.
- Harder: fully offline rendering means not relying on CDN-only assets for core
  content; the generator must vendor/inline what the page needs to work from
  `file://`. We accept this as a generator requirement.
- Harder: no server means no interactive write-back from the rendered site; edits
  are git commits (by design — ADR-0028). Accepted.
- Cost accepted: some conveniences of a hosted app (server-side search, auth-gated
  views) are out of scope; client-side equivalents or git-native alternatives are
  used instead.

## Verification
- [ ] `docs/VISION.md` states the no-hosted-service, static, decentralized goal.
- [ ] The web/issues generators produce a self-contained static directory that
      renders when opened as `file://` (core content needs no network).
- [ ] The generated directory deploys unchanged via file copy / `aws s3 sync` /
      the GitHub Pages workflow; no host-specific coupling in the output.
- [ ] No rendered view requires a running gitrack server; `serve` is preview-only.
