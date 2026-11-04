package web

import (
	"html/template"
	"strings"
)

// rootForHref returns the path-to-site-root prefix for an output href by
// counting its path separators: "docs/x.html" -> "../", "docs/a/b.html" ->
// "../../". This lets a nested page link back to the site root and to sibling
// pages via root-relative hrefs (ADR-0040, ADR-0042).
func rootForHref(href string) string {
	depth := strings.Count(href, "/")
	return strings.Repeat("../", depth)
}

// renderSidebar renders the docs layout tree as a nested <nav class="docs-nav">,
// with each page href prefixed by root (the current page's path-to-site-root)
// and the current page marked. Group headings are plain text; page entries are
// links. Output is escaped via html/template (ADR-0038 safety).
func renderSidebar(tree []*navNode, root, currentHref string) template.HTML {
	var b strings.Builder
	b.WriteString(`<nav class="docs-nav">`)
	writeNavList(&b, tree, root, currentHref)
	b.WriteString(`</nav>`)
	return template.HTML(b.String()) //nolint:gosec // labels/hrefs escaped below; structure is ours
}

// writeNavList writes a <ul> of nav nodes, recursing into children.
func writeNavList(b *strings.Builder, nodes []*navNode, root, currentHref string) {
	if len(nodes) == 0 {
		return
	}
	b.WriteString("<ul>")
	for _, n := range nodes {
		b.WriteString("<li>")
		writeNavNode(b, n, root, currentHref)
		writeNavList(b, n.Children, root, currentHref)
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
}

// writeNavNode writes a single node's label: a link for a page entry (marked
// current when it is the current page), plain text for a group heading.
func writeNavNode(b *strings.Builder, n *navNode, root, currentHref string) {
	label := template.HTMLEscapeString(n.Label)
	if !n.isPage() {
		b.WriteString(`<span class="docs-group">`)
		b.WriteString(label)
		b.WriteString(`</span>`)
		return
	}
	if n.Href == currentHref {
		b.WriteString(`<span class="docs-current">`)
		b.WriteString(label)
		b.WriteString(`</span>`)
		return
	}
	b.WriteString(`<a href="`)
	b.WriteString(template.HTMLEscapeString(root + n.Href))
	b.WriteString(`">`)
	b.WriteString(label)
	b.WriteString(`</a>`)
}
