# ADR-0014: All derived indexes are a single git-ignored local cache

## Status
accepted

## Context
ADR-0012 made the backref index (`refs-in/`) a *committed* cache; ADR-0013 made
the attribute indexes (`state/`, `type/`) *committed* best-effort caches and had
to introduce elaborate `fsck` reconciliation and a `post-merge` hook because a
committed index travels through merges and can desync (a track ending up in two
state buckets after a divergent move — proven in `tmp/index-merge.sh`).

The stakeholder observed the simplifying truth: the entry-point/index subsystem
has **no hard dependency** — it is entirely rebuildable from the authoritative
track objects, and is best thought of as a plain **cache**, implementation
possibly OS-dependent (symlinks where available, plain files otherwise). It
should live under `.gitrack/cache/`, be listed in `.gitignore`, and be updated
on every mutating command and by `fsck`.

We verified (`tmp/cache-gitignore.sh`): a `/cache/` entry in `.gitrack/.gitignore`
keeps `.gitrack/cache/` fully untracked while track objects and config stay
tracked; both symlink and plain-file cache entries work.

The key consequence: a git-ignored cache **never enters a commit, so it never
travels through a merge.** The entire merge-desync hazard of ADR-0013 becomes
*structurally impossible* rather than something to detect and repair.

## Decision
The committed tree contains **only authoritative data**: track objects
(`.gitrack/<uuid>/...`), their forward relationship links, and `.gitrack/config/`.
**Every derived index is a single local cache under `.gitrack/cache/`**, which is
git-ignored. This supersedes ADR-0012's committed `refs-in/` and ADR-0013's
committed `state/`/`type/` indexes:

```
.gitrack/
├── .gitignore            # contains: /cache/
├── config/               # authoritative
├── <uuid>/               # authoritative track objects + forward links
└── cache/                # git-ignored, local, rebuildable — NOT committed
    ├── state/<state>/<uuid>
    ├── type/<kind>/<uuid>
    ├── refs-in/<uuid>/<rel>/<referrer-uuid>   # backrefs move here too
    └── ...                                     # any future derived index
```

- The cache is a pure function of local HEAD's authoritative data. Each clone
  builds its own; nothing about the cache is shared or merged.
- Cache entries MAY be symlinks back to the track dir (fast, self-describing) or
  plain files, chosen per-OS at runtime. Readers must not assume which.
- The cache is (re)built/updated by every mutating command
  (`create|update|edit|close`) and fully rebuilt by `gitrack fsck`.
- `fsck` no longer reconciles a *shared* index across a merge (there is none);
  it simply regenerates the local cache from the authoritative objects. The
  `post-merge` hook's job shrinks to "rebuild the local cache" — a safe,
  conflict-free, local operation.
- The authoritative field remains the only source of truth; the cache is never
  consulted when it could disagree, and a missing/stale cache only costs speed.

## Consequences
- Easier: the move-desync merge hazard is gone — a git-ignored cache is never
  merged. ADR-0013's reconciliation complexity is deleted, not maintained.
- Easier: the committed tree is purely authoritative, which is the cleanest
  "smart data" story — one source of truth, zero committed derived data.
- Easier: the cache can exploit OS features (symlinks) without portability
  worry, since it is never shared.
- Harder: a fresh clone has no cache, so the first list-by-state is a scan (or an
  implicit cache build). We accept this; a one-time `gitrack fsck` (or lazy
  build on first query) makes it fast, and nothing is incorrect meanwhile.
- Harder: the cache is per-working-tree; a bare repo or a server-side tool must
  build its own. Fine — it is cheap and local.
- Cost accepted: cache entries are recomputed per clone rather than transferred.
  Negligible versus the merge-correctness and simplicity gains.

## Verification
- [ ] `.gitrack/.gitignore` contains `/cache/`; `.gitrack/cache/` is untracked
      while track objects and config remain tracked.
- [ ] `state/`, `type/`, and `refs-in/` live ONLY under `.gitrack/cache/`; no
      derived index is committed.
- [ ] A mutating command updates the cache; `gitrack fsck` rebuilds it entirely
      from authoritative objects.
- [ ] A merge never conflicts on or desyncs the cache (the cache is not in git).
- [ ] A reader with no cache still returns correct results (by scanning) and can
      build the cache to go fast; the cache is never authoritative over a field.
