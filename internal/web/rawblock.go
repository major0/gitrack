package web

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// kindRawBlock is the node kind of a block whose HTML was produced by gitrack
// itself rather than by goldmark.
var kindRawBlock = ast.NewNodeKind("RawBlock")

// rawBlock is a block-level node holding finished HTML. Transformers use it to
// replace or add a block whose rendering goldmark does not do: a highlighted code
// fence (ADR-0045) or a table of contents (ADR-0046).
//
// The HTML is written out verbatim, so it must only ever come from gitrack's own
// generators, which escape any author-supplied text they include. It is never
// built from raw Markdown source.
type rawBlock struct {
	ast.BaseBlock
	html string
}

// newRawBlock wraps already-safe HTML for insertion into the tree.
func newRawBlock(html string) *rawBlock { return &rawBlock{html: html} }

// Kind implements ast.Node.
func (*rawBlock) Kind() ast.NodeKind { return kindRawBlock }

// Dump implements ast.Node.
func (n *rawBlock) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// rawBlockRenderer writes a rawBlock's HTML.
type rawBlockRenderer struct{}

// RegisterFuncs implements renderer.NodeRenderer.
func (rawBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindRawBlock, func(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString(n.(*rawBlock).html)
		}
		return ast.WalkSkipChildren, nil
	})
}
