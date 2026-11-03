package web

import (
	"html/template"
	"regexp"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// crossRepoToken matches a `#<repo-url>?track=<uuid>` cross-repo reference
// (ADR-0034). The URL part allows the git-URL charset (letters, digits, and
// :/._@~-) up to the "?track=" marker; the uuid part is hex + hyphens. Tried
// before the local-handle form. The pattern is written to match either
// already-HTML-escaped text (the raw-text autolink path) or raw text (the
// Markdown AST path): it never contains `<`, `>`, or `&`, so escaping does not
// affect it.
var crossRepoToken = regexp.MustCompile(`#([A-Za-z][A-Za-z0-9+.@:/_~-]*\?track=[0-9a-fA-F][0-9a-fA-F-]*)`)

// refToken matches a `#<handle>` LOCAL reference where <handle> is the reference
// charset (ADR-0011): hex (short-uuid/uuid), kebab slug, and UUID hyphens —
// [0-9a-z-].
var refToken = regexp.MustCompile(`#([0-9a-z][0-9a-z-]*)`)

// linkTarget is the result of resolving a `#ref` token: the href and the label
// to show. external marks a cross-repo link (absolute href, no root prefix).
type linkTarget struct {
	href  string
	label string
	// external is true for a cross-repo link whose href is an absolute URL and
	// must not be prefixed with the page's path-to-root.
	external bool
}

// resolveRefToken classifies a single `#...` token (including the leading `#`)
// and, if it is a resolvable reference, returns its link target. ok is false
// when the token is not a reference or does not resolve, in which case the
// caller leaves the token as plain text (ADR-0030 resolve-gated rule).
//
// root is the page's path-to-site-root for LOCAL links; cross-repo links ignore
// it (they are absolute). This is the single tokenizer+resolver shared by both
// the raw-text autolink path and the Markdown AST transform (ADR-0038).
func resolveRefToken(repo *core.Repo, root, token string) (linkTarget, bool) {
	// Cross-repo first, so a URL token is not mis-parsed as a local handle.
	if m := crossRepoToken.FindStringSubmatch(token); m != nil && m[0] == token {
		ref, ok := core.ParseCrossRepo(token)
		if !ok {
			return linkTarget{}, false
		}
		return linkTarget{
			href:     ref.RepoURL,
			label:    "#" + ref.RepoLabel() + "?track=" + ref.ShortUUID(),
			external: true,
		}, true
	}
	if m := refToken.FindStringSubmatch(token); m != nil && m[0] == token {
		handle := token[1:]
		id, err := repo.Resolve(handle)
		if err != nil {
			return linkTarget{}, false
		}
		return linkTarget{href: root + "track/" + id + ".html", label: token}, true
	}
	return linkTarget{}, false
}

// autolink HTML-escapes raw text, then links resolvable references (ADR-0030/
// 0034) for NON-Markdown fields (issue descriptions, comment bodies). Markdown
// sections use renderMarkdown instead, which shares resolveRefToken. Behavior
// of this path is unchanged by the ADR-0038 refactor.
//
//   - a cross-repo `#<repo-url>?track=<uuid>` becomes an external link labeled
//     with the repo + short uuid (no fetch — offline-safe);
//   - a local `#<handle>` that RESOLVES becomes a relative link to that track's
//     page; non-matching / non-resolving tokens stay plain escaped text.
func autolink(repo *core.Repo, root, text string) template.HTML {
	escaped := template.HTMLEscapeString(text)
	escaped = crossRepoToken.ReplaceAllStringFunc(escaped, func(tok string) string {
		t, ok := resolveRefToken(repo, root, tok)
		if !ok {
			return tok
		}
		var b strings.Builder
		b.WriteString(`<a href="`)
		b.WriteString(t.href) // from escaped text; safe as an href
		b.WriteString(`">`)
		b.WriteString(template.HTMLEscapeString(t.label))
		b.WriteString("</a>")
		return b.String()
	})
	out := refToken.ReplaceAllStringFunc(escaped, func(tok string) string {
		t, ok := resolveRefToken(repo, root, tok)
		if !ok {
			return tok
		}
		var b strings.Builder
		b.WriteString(`<a href="`)
		b.WriteString(t.href)
		b.WriteString(`">`)
		b.WriteString(t.label) // token is [0-9a-z#-], no escaping needed
		b.WriteString("</a>")
		return b.String()
	})
	return template.HTML(out) //nolint:gosec // escaped first; only parsed/resolved refs add markup (ADR-0030/0034)
}
