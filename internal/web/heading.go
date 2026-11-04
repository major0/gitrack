package web

import (
	"html/template"
	"net/url"
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
type headingAnchorTransformer struct {
	// tocMin is the fewest h2/h3 headings a document needs for a table of
	// contents to be added before its first section. 0 turns it off (ADR-0046).
	tocMin int
}

// Transform implements parser.ASTTransformer.
func (t headingAnchorTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
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
	var toc []tocEntry
	for _, h := range headings {
		// Read the text before the permalink is appended, so neither the id nor
		// the table of contents ever contains the word "Permalink".
		plain := headingText(h, source)
		id := slugs.next(plain)
		h.SetAttributeString("id", []byte(id))
		if t.tocMin > 0 && inTOC(h, doc) {
			toc = append(toc, tocEntry{node: h, level: h.Level, id: id, text: plain})
		}

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

	if t.tocMin > 0 && len(toc) >= t.tocMin {
		first := toc[0].node
		first.Parent().InsertBefore(first.Parent(), first, newRawBlock(renderTOC(toc)))
	}
}

// tocEntry is one heading listed in a table of contents.
type tocEntry struct {
	node  *ast.Heading
	level int
	id    string
	text  string
}

// inTOC reports whether a heading belongs in the table of contents: an h2 or h3
// that is a direct child of the document. The h1 is the page title, h4 and below
// are too fine to list, and a heading inside a quote or list is not page
// structure.
func inTOC(h *ast.Heading, doc *ast.Document) bool {
	return (h.Level == 2 || h.Level == 3) && h.Parent() == ast.Node(doc)
}

// renderTOC builds the table of contents as a labelled navigation list. An h3
// nests under the h2 before it; an h3 with no h2 before it is listed at the top
// level rather than dropped. Text is the heading's visible words, escaped.
func renderTOC(entries []tocEntry) string {
	type item struct {
		entry tocEntry
		kids  []tocEntry
	}
	var top []*item
	for _, e := range entries {
		if e.level == 3 && len(top) > 0 {
			last := top[len(top)-1]
			last.kids = append(last.kids, e)
			continue
		}
		top = append(top, &item{entry: e})
	}

	var b strings.Builder
	b.WriteString("<nav class=\"toc\" aria-label=\"On this page\">\n<p class=\"toc-title\">On this page</p>\n<ul>\n")
	for _, it := range top {
		b.WriteString("<li>" + tocLink(it.entry))
		if len(it.kids) > 0 {
			b.WriteString("\n<ul>\n")
			for _, k := range it.kids {
				b.WriteString("<li>" + tocLink(k) + "</li>\n")
			}
			b.WriteString("</ul>\n")
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n</nav>\n")
	return b.String()
}

// tocLink is a link to a heading, labelled with its text (its id if it has none,
// such as a heading holding only an image with no alt text).
func tocLink(e tocEntry) string {
	label := strings.TrimSpace(e.text)
	if label == "" {
		label = e.id
	}
	frag := (&url.URL{Fragment: e.id}).EscapedFragment()
	return `<a href="#` + template.HTMLEscapeString(frag) + `">` + template.HTMLEscapeString(label) + `</a>`
}
