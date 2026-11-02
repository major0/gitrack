# ADR-0030: Autolink `#refs` in rendered output, conservatively and safely

## Status
accepted

## Context
ADR-0011 defined the `#` reference syntax (`#<uuid>`, `#<short-uuid>`, `#<slug>`,
cross-repo `#<repo-url>?track=<uuid>`) but left *rendering* to the generators.
The web generator stores a track's `description` (and will render comment
bodies) verbatim; nothing turns a `#ref` into a link yet.

Two concerns shape the implementation:

1. **Safety / escaping order.** The content is user-authored text rendered into
   HTML. Injecting `<a>` tags must happen AFTER HTML-escaping the text, so the
   link markup is the only unescaped HTML and a title containing `<script>`
   cannot break out. (ADR-0029 already accepts `template.HTML` for local tracker
   data, but the escape-then-linkify order is still required for correctness.)
2. **False links.** `#` appears in text that is not a reference (a `#heading`,
   a `#` in prose). Linkifying every `#token` would create dead or wrong links.

## Decision
The web generator autolinks a `#<handle>` token **only when it resolves to an
existing track** (via `core.Resolve`); otherwise the token is left as plain
(escaped) text. This keeps non-reference `#`s and dangling refs untouched — no
false or dead links.

Rendering algorithm (`web.autolink(repo, root, text)`):

1. HTML-escape the raw text.
2. Scan the escaped text for `#<handle>` tokens where `<handle>` matches the
   reference charset (hex for short-uuid/uuid, kebab for slug: `[0-9a-z-]` plus
   UUID hyphens).
3. For each, try `core.Resolve(handle)`. On success, replace the token with
   `<a href="{root}track/{uuid}.html">#{handle}</a>` (relative link, ADR-0029).
   On failure (no match or ambiguous), leave the escaped token as-is.
4. Return the result as `template.HTML`.

Scope of the first slice:

- Applies to a track's `description` in the web output; the same `autolink`
  function will render comment bodies when comments are added to the web output.
- **Local refs only** (uuid/short/slug). Cross-repo `#<repo-url>?track=<uuid>`
  links are a documented follow-on; until then such a token simply does not
  resolve locally and stays plain text (no breakage).
- Token-level, not a full Markdown parse. A Markdown renderer is a larger,
  separate decision; conservative token linkifying is correct and safe now.

## Consequences
- Easier: `#ref` in a description becomes a real relative link to the track page,
  working offline and when the site is moved (ADR-0029).
- Safer: escape-before-linkify prevents HTML injection; resolve-gated linking
  prevents false/dead links.
- Harder: short-uuid/slug resolution happens per token at generate time; O(refs ×
  tracks) in the worst case for slug lookups. Fine for generation; can cache a
  handle->uuid map later if needed.
- Cost accepted: cross-repo and full-Markdown rendering are deferred; a `#`
  that is not a resolvable local ref renders as plain text (acceptable).

## Verification
- [ ] A `#<slug>`/`#<short-uuid>`/`#<uuid>` in a description that resolves renders
      as `<a href="...track/<uuid>.html">#...</a>` with a RELATIVE href.
- [ ] A `#` that does not resolve (dangling ref, `#heading`) stays plain text.
- [ ] Text is HTML-escaped before links are injected (a `<` in the body is
      escaped; no injection).
- [ ] Generated links are relative and work from `file://` / a moved site.
