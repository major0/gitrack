package web

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"

	"github.com/major0/gitrack/internal/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// anyRefToken matches either a cross-repo or a local `#...` token in RAW
// (unescaped) Markdown prose. It is the union used to find candidate tokens in
// an ast.Text value; each hit is then classified+resolved by resolveRefToken,
// which re-validates it as a full token. The alternation tries the longer
// cross-repo form first.
var anyRefToken = regexp.MustCompile(
	`#[A-Za-z][A-Za-z0-9+.@:/_~-]*\?track=[0-9a-fA-F][0-9a-fA-F-]*` + // cross-repo
		`|#[0-9a-z][0-9a-z-]*`) // local handle

// renderMarkdown renders a Markdown source string to safe HTML for the theme
// shell (ADR-0038). It is the single Markdown entry point for every Markdown
// section (docs, blog, release notes, portal). #ref autolinking (ADR-0030/0034)
// is applied as an AST transform so it only touches prose text — never code
// spans, code blocks, or existing links — and composes with goldmark's own
// escaping.
//
// root is the page's path-to-site-root for local links. Raw inline/block HTML
// in the source is NOT rendered as markup (goldmark default): untrusted authored
// content cannot inject <script>/<iframe> (ADR-0038 Safety).
func renderMarkdown(repo *core.Repo, root, src string) (template.HTML, error) {
	res, err := renderMarkdownWith(markdownOpts{repo: repo, root: root}, src)
	return res.HTML, err
}

// markdownOpts carries what a render needs beyond the source: the repo for
// #ref resolution, the page's path-to-root, and an optional highlighter for
// fenced code blocks (nil leaves every fence to goldmark's plain rendering).
type markdownOpts struct {
	repo *core.Repo
	root string
	hl   *highlighter
}

// mdResult is rendered Markdown plus the stylesheet its highlighted code needs.
// CSS is empty unless at least one block was highlighted, so a page attaches it
// only when it is used (ADR-0045).
type mdResult struct {
	HTML template.HTML
	CSS  template.CSS
}

// renderMarkdownWith is renderMarkdown with a highlighter for fenced code.
func renderMarkdownWith(o markdownOpts, src string) (mdResult, error) {
	transformers := []util.PrioritizedValue{
		// lower runs first: ids come from the authored heading text, before the
		// autolinker rewrites any #refs inside it
		util.Prioritized(headingAnchorTransformer{}, 50),
		util.Prioritized(&autolinkTransformer{repo: o.repo, root: o.root}, 100),
	}
	fences := &fenceHighlighter{hl: o.hl}
	if o.hl != nil {
		transformers = append(transformers, util.Prioritized(fences, 150))
	}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithASTTransformers(transformers...)),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(
			util.Prioritized(highlightedCodeRenderer{}, 500),
		)),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return mdResult{}, err
	}
	res := mdResult{HTML: template.HTML(buf.String())} //nolint:gosec // goldmark escapes; raw HTML disabled; links added only for resolved refs; highlighted blocks are chroma output, which escapes
	if fences.highlighted > 0 {
		res.CSS = o.hl.css
	}
	return res, nil
}

// autolinkTransformer is a goldmark AST transformer that replaces resolvable
// #ref tokens in prose Text nodes with Link nodes, reusing resolveRefToken
// (ADR-0038). It skips tokens inside code spans, code blocks, and existing
// links so a `#` in code or a URL is never mis-linked.
type autolinkTransformer struct {
	repo *core.Repo
	root string
}

// Transform implements parser.ASTTransformer. It does two passes over the AST:
// rewrite relative `.md` link destinations to `.html` so intra-doc links resolve
// to the generated pages (ADR-0040), and linkify resolvable #ref tokens in prose
// (ADR-0030/0034).
func (tr *autolinkTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	// Collect target Text nodes and Link nodes first; mutating during Walk is
	// unsafe.
	var textTargets []*ast.Text
	var linkTargets []*ast.Link
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Text:
			if !inSkippedContext(node) {
				textTargets = append(textTargets, node)
			}
		case *ast.Link:
			linkTargets = append(linkTargets, node)
		}
		return ast.WalkContinue, nil
	})
	for _, l := range linkTargets {
		l.Destination = rewriteDocLink(l.Destination)
	}
	for _, t := range textTargets {
		tr.rewriteText(t, source)
	}
}

// mdLinkRe matches a relative link destination ending in ".md", optionally with
// a "#fragment" or "?query", that should be rewritten to the generated ".html"
// page (ADR-0040). Absolute URLs (scheme or leading "/") are left untouched.
var mdLinkRe = regexp.MustCompile(`^(.+)\.md(#[^?]*|\?[^#]*|)$`)

// rewriteDocLink rewrites a relative "<path>.md" link destination to
// "<path>.html", preserving any #fragment/?query. An absolute URL (has a scheme
// or starts with "/") or a non-.md link is returned unchanged.
func rewriteDocLink(dest []byte) []byte {
	s := string(dest)
	if s == "" || strings.HasPrefix(s, "/") || hasURLScheme(s) {
		return dest
	}
	m := mdLinkRe.FindStringSubmatch(s)
	if m == nil {
		return dest
	}
	return []byte(m[1] + ".html" + m[2])
}

// hasURLScheme reports whether s begins with a URL scheme like "http:" or
// "mailto:" (so it is an absolute link, not an intra-repo path).
func hasURLScheme(s string) bool {
	i := strings.IndexByte(s, ':')
	if i <= 0 {
		return false
	}
	for j := 0; j < i; j++ {
		c := s[j]
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isSchemeChar := isAlpha || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'
		if j == 0 && !isAlpha {
			return false
		}
		if !isSchemeChar {
			return false
		}
	}
	return true
}

// inSkippedContext reports whether a Text node is inside a context where #refs
// must NOT be linkified: a code span, a fenced/indented code block, or an
// existing link/autolink.
func inSkippedContext(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case ast.KindCodeSpan, ast.KindLink, ast.KindAutoLink,
			ast.KindFencedCodeBlock, ast.KindCodeBlock:
			return true
		}
	}
	return false
}

// rewriteText splits a prose Text node around resolvable #ref tokens, replacing
// it in its parent with a sequence of String (plain) and Link (resolved ref)
// nodes. If the node has no resolvable token it is left untouched.
func (tr *autolinkTransformer) rewriteText(t *ast.Text, source []byte) {
	val := t.Segment.Value(source)
	locs := anyRefToken.FindAllIndex(val, -1)
	if len(locs) == 0 {
		return
	}
	// Build replacement nodes, resolving each candidate; keep non-resolving
	// tokens as plain text (ADR-0030 resolve-gated rule).
	parent := t.Parent()
	if parent == nil {
		return
	}
	var replacements []ast.Node
	last := 0
	linkedAny := false
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		target, ok := tr.resolve(string(val[start:end]))
		if !ok {
			continue // leave this token embedded in surrounding text
		}
		if start > last {
			replacements = append(replacements, ast.NewString(cloneBytes(val[last:start])))
		}
		link := ast.NewLink()
		link.Destination = []byte(target.href)
		link.AppendChild(link, ast.NewString(cloneBytes([]byte(target.label))))
		replacements = append(replacements, link)
		last = end
		linkedAny = true
	}
	if !linkedAny {
		return
	}
	if last < len(val) {
		replacements = append(replacements, ast.NewString(cloneBytes(val[last:])))
	}
	for _, r := range replacements {
		parent.InsertBefore(parent, t, r)
	}
	parent.RemoveChild(parent, t)
}

// resolve classifies and resolves a candidate token to a link target, reusing
// the shared resolver (ADR-0038).
func (tr *autolinkTransformer) resolve(token string) (linkTarget, bool) {
	return resolveRefToken(tr.repo, tr.root, token)
}

// cloneBytes returns a copy so inserted String nodes do not alias the source
// slice.
func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
