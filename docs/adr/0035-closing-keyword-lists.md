# ADR-0035: A closing keyword governs the whole list of refs that follows

## Status
accepted

## Context
ADR-0031/0033 recognized a commit-message directive as a single `<verb> #<handle>`
pair, classified by the word immediately before each ref. The stakeholder wants
the GitHub/GitLab behavior, and specifically GitLab's: a keyword should close
EVERY issue in the list that follows it, including comma/"and"-separated lists —
`Closes #a #b #c`, `Resolves #a, #b, #c`, `Closes #a, #b, and #c`.

Research (rephrased from the vendors' docs for compliance):

- **GitHub** closing keywords: close, closes, closed, fix, fixes, fixed, resolve,
  resolves, resolved. For MULTIPLE issues GitHub requires the keyword before each
  ref (`fixes #4, resolve #6`); a bare list after one keyword does not close all.
  (docs.github.com, linking-a-pull-request-to-an-issue)
- **GitLab** default closing pattern closes a LIST after one keyword: its regex
  repeats `%{issue_ref}` with separators `, *`, ` +and +`. Keywords: close(s/d/
  ing), fix(es/ed/ing), resolve(s/d/ing), implement(s/ed/ing), with an optional
  trailing colon. (docs.gitlab.com, issue_closing_pattern)

The GitLab list behavior is a strict superset of GitHub's: "keyword before each"
still works because each ref is also the first of its own (one-element) list.
Adopting the list behavior satisfies the request and both platforms' idioms, and
suits Conventional Commits footers (`Closes #a #b` capitalized at the end).

## Decision
A commit/comment-message directive parser (`core.ParseDirectives`,
`core.classifyMention`) attributes a verb to each `#<handle>` by **run**, not by
the single preceding word:

- scan left to right; a **closing keyword** (close/closes/closed, fix/fixes/
  fixed, resolve(s/d), resolving, implement(s/ed/ing)), with an optional trailing
  `:`, opens a "closes" run; a **relates keyword** (relate(s/d), related to,
  relates to) opens a "relates" run.
- within a run, consecutive `#<handle>` tokens separated ONLY by whitespace,
  commas, or the word "and" all receive the run's verb. Any other word (or a
  blank line) ends the run.
- a `#<handle>` not in any run is a plain "mention".
- keyword matching is case-insensitive, so capitalized footer forms (`Closes`,
  `Resolves`) are recognized.

So `Closes #a #b #c`, `Resolves #a, #b, #c`, and `Closes #a, #b, and #c` all
attribute `closes` to every listed track; `fixes #4, resolve #6` still works
(each keyword opens its own run). `git track commit` then closes every track a
`closes`-run names (ADR-0033), and `Mentions` (ADR-0031) classifies each
accordingly.

## Consequences
- Easier: one keyword closes a whole list — the GitLab idiom and the stakeholder's
  request; GitHub's keyword-before-each still works (superset).
- Easier: Conventional-Commit footers (`Closes #a #b`) work as written.
- Harder: the parser is a small state machine (run tracking) rather than a
  per-token lookback; more logic, but exact and testable.
- Cost accepted: an aggressive run could over-capture if a list is interrupted by
  a token that merely looks like a separator; the rule ends a run on any
  non-(ws/comma/"and") token, which matches the vendors' behavior.

## Verification
- [ ] `Closes #a #b #c` attributes closes to a, b, and c.
- [ ] `Resolves #a, #b, #c` and `Closes #a, #b, and #c` attribute closes to all.
- [ ] `fixes #4, resolve #6` still attributes closes to both (keyword-before-each).
- [ ] A ref after a non-keyword word (`see #x`) is a mention, and a run ends at
      an interrupting word (`closes #a then wip #b` -> a closes, b mentions).
- [ ] `git track commit` closes every track named in a closes-run.
- [ ] Keyword matching is case-insensitive (footer `Closes`/`Resolves`).
