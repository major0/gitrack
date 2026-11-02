# ADR-0024: `git track serve` — loopback live preview server

## Status
accepted

## Context
The stakeholder wants a `git track serve` command in the spirit of `hugo
server`: bind a random port, serve HTTP for live monitoring of the tracker, and
eventually host both the Git-Web source-code view and the Git-Track tracks view.

Two concerns must be pinned:

1. **Security of the bind.** A dev preview server must never be reachable off the
   machine. It binds **loopback only** on a random port (`127.0.0.1:0`), and
   prints the resulting `http://127.0.0.1:<port>` URL — matching Hugo's localhost
   default and the project's rule that any local HTTP server binds explicitly to
   loopback.
2. **Scope.** The full ask (source browser + tracks view + live reload) is large,
   and the static `web`/`issues` generators do not exist yet. Per prove-the-
   method-first, this ADR defines the command and its security contract, and the
   first implementation is a bounded slice; the rest is explicitly deferred.

## Decision
`git track serve [--addr 127.0.0.1:0]` starts an HTTP server bound to **loopback
only** on, by default, a random port, and prints the chosen URL. It renders the
**current tracker state live**: each request reads the authoritative tree
(`core`) at request time, so the view always reflects the working tree with no
rebuild step — the "live monitoring" property. It is read-only and has no
write/mutation endpoints.

Deliberately bounded for the first slice:

- Serves the **tracks view** (list of tracks, and a per-track page) rendered from
  `core`. The Git-Web source-code view is a documented follow-on.
- "Live" is achieved by reading on each request (no cache of rendered HTML).
  WebSocket/SSE live-reload-on-change is a follow-on; re-requesting always shows
  current state.
- `--addr` may override the bind for advanced use, but the value is validated to
  a loopback host; a non-loopback bind is refused. The default is `127.0.0.1:0`.

The rendered HTML is produced by the same view logic the static `web` generator
(ADR-0010, forthcoming) will use, so `serve` and the published site stay
consistent — `serve` is the live view, the generator is the frozen snapshot.

## Consequences
- Easier: a one-command live view of the tracker, like `hugo server`, for local
  work and demos.
- Easier: reading live per request means no staleness and no watcher to maintain
  in the first slice.
- Safer: loopback-only bind means the server is never exposed to the network;
  the non-loopback refusal makes that a hard guarantee, not a default.
- Harder: rendering per request is O(tracks) each hit; fine for local use, and a
  cache can be added later if needed.
- Deferred (named, not built): the Git-Web source browser, live-reload push, and
  sharing the renderer with the static generator (which itself is forthcoming).

## Verification
- [ ] `git track serve` binds `127.0.0.1` on a random port and prints the URL.
- [ ] The server is not reachable on a non-loopback interface; a non-loopback
      `--addr` is refused.
- [ ] A request renders the current tracker state read live from the tree (a
      track created after startup appears on the next request without restart).
- [ ] The server exposes no mutation endpoint (read-only).
- [ ] Shutdown is clean on SIGINT.
