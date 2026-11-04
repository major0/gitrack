# References and autolinking

A reference points at a track. Because tracks are keyed by UUID — not by a
typeable number — gitrack accepts several human-friendly forms that all resolve
to the same track, and autolinks them in rendered output.

## The `#ref` syntax

A reference starts with `#` and resolves to a track UUID:

| form | resolves to | notes |
|---|---|---|
| `#<uuid>` | that UUID | the full canonical id |
| `#<short-uuid>` | a UUID | shortest unique hex prefix (git-short-hash style); an ambiguous prefix errors with the candidates |
| `#<slug>` | a UUID | the authored slug or one of its aliases |
| `#<repo-url>?track=<uuid>` | a remote UUID | a cross-repo reference (the uuid may be short) |

Slug versus short-uuid is disambiguated by shape: a pure-hex token is a
short-uuid; a slug may not be pure hex (enforced when the slug is minted). So
`#a1b2c3` is a short id and `#new-login-flow` is a slug, unambiguously.

## Slugs and aliases

Every track gets a repo-unique **slug** derived from its title at creation,
editable later. The slug is the handle you type in commands, references, and
URLs; the UUID stays canonical underneath. When you retitle a track and its slug
changes, the old slug is kept as an **alias** so existing references keep
resolving.

```sh
git track show new-login-flow          # reference by slug
git track show a1b2c3                   # reference by short id
git track show 0192f1a2-7c3e-7abc-...   # reference by full uuid
```

## Where references appear

- **In descriptions and comments.** Write `#new-login-flow` in a track's body or
  a comment; the web and issues generators autolink it.
- **In task lists.** `- [ ] #new-login-flow landed` is a dependency-backed
  checklist item: its checkbox renders from the referenced track's state, and it
  also materializes a `depends-on` edge. See [Relationships](relationships.md).
- **In commit messages.** `Closes #new-login-flow` closes the track when you
  commit through gitrack. See [Commit integration](commit-integration.md).

## How references render (autolinking)

In the generated website, a `#<ref>` is turned into a link **only when it
resolves** to an existing track; a non-resolving `#token` (a `#heading`, a stray
`#`) stays plain text, so there are no dead or false links.

- A **local** ref becomes a relative link to that track's page, so it works
  offline and from a moved or copied site.
- A **cross-repo** ref `#<repo-url>?track=<uuid>` becomes an external link to the
  remote repository, labeled with the repo name and short uuid. It is parsed and
  linked with **no network fetch** — rendering stays offline and trust-safe.
  Reading the remote track's title or state would require a fetch and is
  deliberately out of the render path.

Autolinking is consistent across every part of the site that renders authored
text — issue descriptions, comments, docs pages, blog posts, release notes — not
just issues. Inside code spans and fenced code blocks, a `#ref` is left
untouched, so documentation that shows a literal `#tag` is never mangled.

## The commit trailer

From a code commit, the `Gitrack-Ref: <ref>` trailer carries the same value with
the `#` dropped. It is one of the signals gitrack uses to compute a track's
`referenced-by` (the other being commits that touched the track's directory).
See [Commit integration](commit-integration.md).
