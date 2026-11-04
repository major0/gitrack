package web

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// fallbackHeadingID names a heading with nothing slug-able in it (only
// punctuation or emoji).
const fallbackHeadingID = "section"

// githubSlug turns heading text into an id the way GitHub does, so a link such
// as `#getting-started` written for a README on GitHub also works on the site:
// lower-case; keep letters, marks, numbers, '_' and '-'; each space becomes '-';
// drop everything else. It is deliberately not goldmark's built-in generator,
// which discards all non-ASCII text and rewrites '_' (ADR-0044).
func githubSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-', r == '_',
			unicode.IsLetter(r), unicode.IsMark(r), unicode.IsNumber(r),
			unicode.Is(unicode.Pc, r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// slugger hands out unique ids within one document. A repeated heading gets
// -1, -2, ... appended, GitHub's rule, and the counter is per base slug.
type slugger struct {
	used  map[string]bool
	count map[string]int
}

func newSlugger() *slugger {
	return &slugger{used: map[string]bool{}, count: map[string]int{}}
}

// next returns a unique id for a heading's text.
func (s *slugger) next(headingText string) string {
	base := githubSlug(headingText)
	if base == "" {
		base = fallbackHeadingID
	}
	id := base
	for s.used[id] {
		s.count[base]++
		id = base + "-" + strconv.Itoa(s.count[base])
	}
	s.used[id] = true
	return id
}

// headingText is the visible text of a heading: its text and string nodes,
// descending through code spans, emphasis, and links, so markup characters and
// link destinations never reach the id.
func headingText(h *ast.Heading, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(h, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// headingAnchorTransformer gives every heading an id and appends a permalink to
// it (ADR-0044). It runs before the autolink transformer so ids come from the
// authored text.
type headingAnchorTransformer struct{}

// Transform implements parser.ASTTransformer.
func (headingAnchorTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	// Collect first; appending children during Walk is unsafe.
	var headings []*ast.Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering {
			headings = append(headings, h)
		}
		return ast.WalkContinue, nil
	})
	slugs := newSlugger()
	for _, h := range headings {
		id := slugs.next(headingText(h, source))
		h.SetAttributeString("id", []byte(id))

		// The link text is real words so the link has an accessible name; CSS
		// hides the word and shows a pilcrow instead. goldmark does not let
		// aria-label through on a link, so the text carries the name.
		link := ast.NewLink()
		link.Destination = []byte("#" + id)
		link.SetAttributeString("class", []byte("anchor"))
		link.SetAttributeString("title", []byte("Permalink to this section"))
		link.AppendChild(link, ast.NewString([]byte("Permalink")))
		h.AppendChild(h, link)
	}
}
