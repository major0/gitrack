package web

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// fenceHighlighter is an AST transformer that highlights fenced code blocks by
// the language named in their info string (```go). A block with no language, an
// unknown language, or that cannot be highlighted within the limits keeps its
// plain rendering. It runs after the autolinker, which already leaves code
// blocks untouched, so a #ref inside code is never linked.
type fenceHighlighter struct {
	hl          *highlighter
	highlighted int // how many blocks were replaced; the page needs the CSS if > 0
}

// Transform implements parser.ASTTransformer.
func (f *fenceHighlighter) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	// Collect first; replacing nodes during Walk is unsafe.
	var blocks []*ast.FencedCodeBlock
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if b, ok := n.(*ast.FencedCodeBlock); ok && entering {
			blocks = append(blocks, b)
		}
		return ast.WalkContinue, nil
	})
	for _, b := range blocks {
		lang := string(b.Language(source))
		lexer := lexerByName(lang)
		if lexer == nil {
			continue
		}
		var code bytes.Buffer
		lines := b.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			code.Write(seg.Value(source))
		}
		// Invalid UTF-8 would become the replacement character in the plain
		// renderer too; do it up front so chroma sees valid text.
		out, ok := f.hl.block(lexer, lang, strings.ToValidUTF8(code.String(), "\uFFFD"))
		if !ok {
			continue
		}
		parent := b.Parent()
		if parent == nil {
			continue
		}
		parent.ReplaceChild(parent, b, newRawBlock(out))
		f.highlighted++
	}
}
