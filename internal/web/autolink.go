package web

import (
	"html/template"
	"regexp"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// refToken matches a `#<handle>` reference where <handle> is the reference
// charset (ADR-0011): hex (short-uuid/uuid), kebab slug, and UUID hyphens —
// [0-9a-z-]. Matched against ALREADY-ESCAPED text, so it cannot span injected
// markup. A trailing hyphen or over-long run is still passed to core.Resolve,
// which rejects non-resolving handles.
var refToken = regexp.MustCompile(`#([0-9a-z][0-9a-z-]*)`)

// autolink HTML-escapes text, then turns each `#<handle>` that RESOLVES to an
// existing track into a relative link to that track's page (ADR-0030). Tokens
// that do not resolve (dangling refs, non-reference '#') are left as plain
// escaped text. root is the page's relative path to the site root so links work
// from file:// and any subpath.
func autolink(repo *core.Repo, root, text string) template.HTML {
	escaped := template.HTMLEscapeString(text)
	out := refToken.ReplaceAllStringFunc(escaped, func(tok string) string {
		handle := tok[1:] // drop the leading '#'
		id, err := repo.Resolve(handle)
		if err != nil {
			return tok // not a resolvable local ref -> leave as plain text
		}
		// tok is already escaped and handle is [0-9a-z-]; safe to embed.
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
	return template.HTML(out) //nolint:gosec // escaped first; only resolved refs add markup (ADR-0030)
}
