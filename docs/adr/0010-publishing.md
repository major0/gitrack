# ADR-0010: Publish a static site on every commit that touches the tracker

## Status
accepted

## Context
The stakeholder wants to disable a project's GitHub issue tracker and link into a
per-project GitHub Pages site instead, regenerated on every commit in which
`.gitrack/` changed. We confirmed the mechanism exists: the official
`actions/upload-pages-artifact` + `actions/deploy-pages` actions deploy a static
site on `push`, and a `paths:` filter scopes the trigger to tracker changes.

The site must be a pure function of the on-disk format (no server, no database),
consistent with the derived-data rule (ADR-0009): the generator reads authored
tracker files and computes every view.

## Decision
We will ship a GitHub Actions workflow (`.github/workflows/gitrack-pages.yml`)
that triggers `on: push` to the default branch with `paths: ['.gitrack/**']`,
runs the gitrack static-site generator over the repo, uploads the output as a
Pages artifact, and deploys it with `actions/deploy-pages`. The workflow is the
shipped default a project copies in; the generator it invokes (`gitrack web` /
`gitrack issues`, later work) turns the format into HTML. Publishing is
idempotent and derived — re-running on the same tree yields the same site.

The generator consumes git history for the derived `referenced-by` view
(ADR-0009), so the workflow checks out with full history (`fetch-depth: 0`).

## Consequences
- Easier: a project disables hosted issues and links to its Pages site; the
  tracker lives entirely in-repo and republishes automatically.
- Easier: the path filter means unrelated code pushes don't rebuild the site.
- Harder: the site is read-only. Creating/editing a track is a commit, not a web
  form — by design (edits must merge with code). We accept this; write-back from
  a static page is out of scope.
- Harder: `fetch-depth: 0` makes the checkout heavier on large repos. Acceptable;
  `referenced-by` needs history. A shallow fallback can drop that view.
- Cost accepted: the workflow is GitHub-specific. A GitLab Pages equivalent is a
  later, parallel artifact; the generator itself is host-agnostic.

## Verification
- [ ] `.github/workflows/gitrack-pages.yml` triggers on push with
      `paths: ['.gitrack/**']` and deploys via `actions/deploy-pages`.
- [ ] The workflow checks out full history (`fetch-depth: 0`).
- [ ] The generator produces the same site from the same tree (idempotent) and
      reads only authored tracker files plus git history.
- [ ] A commit touching only code (not `.gitrack/`) does not trigger a rebuild.
