package web

import (
	"html/template"
	"regexp"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// crossRepoToken matches a `#<repo-url>?track=<uuid>` cross-repo reference
// (ADR-0034) on ALREADY-ESCAPED text. The URL part allows the git-URL charset
// (letters, digits, and :/._@~-) up to the "?track=" marker; the uuid part is
// hex + hyphens. Tried before the local-handle form.
var crossRepoToken = regexp.MustCompile(`#([A-Za-z][A-Za-z0-9+.@:/_~-]*\?track=[0-9a-fA-F][0-9a-fA-F-]*)`)

// refToken matches a `#<handle>` LOCAL reference where <handle> is the reference
// charset (ADR-0011): hex (short-uuid/uuid), kebab slug, and UUID hyphens —
// [0-9a-z-]. Matched against ALREADY-ESCAPED text, so it cannot span injected
// markup.
var refToken = regexp.MustCompile(`#([0-9a-z][0-9a-z-]*)`)

// autolink HTML-escapes text, then links references (ADR-0030/0034):
//   - a cross-repo `#<repo-url>?track=<uuid>` becomes an external link to the
//     remote repo, labeled with the repo + short uuid (no fetch — offline-safe);
//   - a local `#<handle>` that RESOLVES becomes a relative link to that track's
//     page.
//
// Non-matching / non-resolving tokens stay plain escaped text. root is the
// page's relative path to the site root so local links work from file://.
func autolink(repo *core.Repo, root, text string) template.HTML {
	escaped := template.HTMLEscapeString(text)
	// Cross-repo first, so a URL token is not mis-parsed as a local handle.
	escaped = crossRepoToken.ReplaceAllStringFunc(escaped, func(tok string) string {
		ref, ok := core.ParseCrossRepo(tok)
		if !ok {
			return tok
		}
		// ref.RepoURL came from the escaped text; it is safe to embed as an href.
		var b strings.Builder
		b.WriteString(`<a href="`)
		b.WriteString(ref.RepoURL)
		b.WriteString(`">#`)
		b.WriteString(template.HTMLEscapeString(ref.RepoLabel()))
		b.WriteString(`?track=`)
		b.WriteString(ref.ShortUUID())
		b.WriteString("</a>")
		return b.String()
	})
	// Then local handles (skip anything already inside an injected <a>).
	out := refToken.ReplaceAllStringFunc(escaped, func(tok string) string {
		handle := tok[1:]
		id, err := repo.Resolve(handle)
		if err != nil {
			return tok
		}
		var b strings.Builder
		b.WriteString(`<a href="`)
		b.WriteString(root)
		b.WriteString("track/")
		b.WriteString(id)
		b.WriteString(`.html">`)
		b.WriteString(tok)
		b.WriteString("</a>")
		return b.String()
	})
	return template.HTML(out) //nolint:gosec // escaped first; only parsed/resolved refs add markup (ADR-0030/0034)
}
