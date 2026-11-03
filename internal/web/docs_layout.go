package web

import (
	"sort"
	"strings"
)

// navNode is one node in the docs sidebar tree (ADR-0040): either a page entry
// (Href set, linking to a doc) or a group heading (Href empty, Children set).
// A page entry may also have children (a section landing page with subpages).
type navNode struct {
	Label    string
	Href     string // root-relative output href (e.g. "docs/intro.html"); "" for a pure group
	Children []*navNode
}

// isPage reports whether the node links to a doc.
func (n *navNode) isPage() bool { return n.Href != "" }

// buildLayout produces the docs sidebar tree. If a layout file was authored
// (layoutSrc non-empty) it is parsed and reconciled against the docs on disk;
// otherwise an auto layout is generated from the doc tree (ADR-0040).
//
// docs maps a doc's rel path (no extension, slash-separated) to its resolved
// title; order is the sorted rel-path list for deterministic auto layout.
func buildLayout(layoutSrc string, docs map[string]string, order []string) []*navNode {
	if strings.TrimSpace(layoutSrc) == "" {
		return autoLayout(docs, order)
	}
	return reconcileLayout(parseLayout(layoutSrc, docs), docs, order)
}

// docHref is the fixed output href for a doc rel path (ADR-0040: output prefix
// is always "docs/" regardless of the source dir name).
func docHref(rel string) string { return "docs/" + rel + ".html" }

// parseLayout parses an authored layout file into a nav tree using indentation
// for nesting (two spaces or one tab per level). A token resolving to a known
// doc rel path becomes a page entry; otherwise the line is a group heading. An
// optional ": label" overrides the derived label.
func parseLayout(src string, docs map[string]string) []*navNode {
	type stackEnt struct {
		node  *navNode
		depth int
	}
	var roots []*navNode
	var stack []stackEnt

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		depth := indentDepth(line)
		token, label := splitLayoutLine(strings.TrimSpace(line))
		node := layoutNode(token, label, docs)

		// Pop to the parent at a shallower depth.
		for len(stack) > 0 && stack[len(stack)-1].depth >= depth {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, node)
		}
		stack = append(stack, stackEnt{node: node, depth: depth})
	}
	return roots
}

// layoutNode builds a nav node for a layout token: a page entry if the token
// names a known doc (with or without a .md suffix), else a group heading. The
// label is the explicit one, else the doc title, else the token itself.
func layoutNode(token, label string, docs map[string]string) *navNode {
	rel := strings.TrimSuffix(token, ".md")
	if title, ok := docs[rel]; ok {
		l := label
		if l == "" {
			l = title
		}
		return &navNode{Label: l, Href: docHref(rel)}
	}
	l := label
	if l == "" {
		l = token
	}
	return &navNode{Label: l} // group heading (unresolved => dead-safe label)
}

// indentDepth returns the nesting depth from leading whitespace: one tab or two
// spaces per level.
func indentDepth(line string) int {
	depth := 0
	i := 0
	for i < len(line) {
		if line[i] == '\t' {
			depth++
			i++
			continue
		}
		if strings.HasPrefix(line[i:], "  ") {
			depth++
			i += 2
			continue
		}
		break
	}
	return depth
}

// splitLayoutLine splits "token: label" into its token and optional label.
func splitLayoutLine(s string) (token, label string) {
	if i := strings.Index(s, ":"); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

// reconcileLayout appends any docs not referenced in the authored layout under
// an "Other" group so nothing on disk is unreachable (ADR-0040).
func reconcileLayout(roots []*navNode, docs map[string]string, order []string) []*navNode {
	referenced := map[string]bool{}
	var mark func(ns []*navNode)
	mark = func(ns []*navNode) {
		for _, n := range ns {
			if n.isPage() {
				referenced[n.Href] = true
			}
			mark(n.Children)
		}
	}
	mark(roots)

	var missing []*navNode
	for _, rel := range order {
		if !referenced[docHref(rel)] {
			missing = append(missing, &navNode{Label: docs[rel], Href: docHref(rel)})
		}
	}
	if len(missing) > 0 {
		roots = append(roots, &navNode{Label: "Other", Children: missing})
	}
	return roots
}

// autoLayout generates a sidebar from the doc tree: directories become groups,
// files become entries, ordered alphabetically with index/README first within
// each directory (ADR-0040).
func autoLayout(docs map[string]string, order []string) []*navNode {
	// Group docs by their immediate directory ("" for top level).
	root := &navNode{}
	groups := map[string]*navNode{"": root}

	// Ensure group nodes exist for every directory prefix, in order.
	ensureGroup := func(dir string) *navNode {
		if g, ok := groups[dir]; ok {
			return g
		}
		// Create nested groups along the path.
		parts := strings.Split(dir, "/")
		cur := root
		path := ""
		for _, p := range parts {
			if path == "" {
				path = p
			} else {
				path += "/" + p
			}
			g, ok := groups[path]
			if !ok {
				g = &navNode{Label: p}
				cur.Children = append(cur.Children, g)
				groups[path] = g
			}
			cur = g
		}
		return cur
	}

	sorted := append([]string(nil), order...)
	sort.Slice(sorted, func(i, j int) bool { return autoLess(sorted[i], sorted[j]) })
	for _, rel := range sorted {
		dir, base := splitRel(rel)
		g := ensureGroup(dir)
		node := &navNode{Label: docs[rel], Href: docHref(rel)}
		if base == "index" || base == "README" {
			// Directory landing: make it the group's own link, first.
			if g != root {
				g.Href = docHref(rel)
				continue
			}
		}
		g.Children = append(g.Children, node)
	}
	return root.Children
}

// autoLess orders doc rel paths: shallower paths first, index/README before
// siblings, then lexicographic.
func autoLess(a, b string) bool {
	da, ba := splitRel(a)
	db, bb := splitRel(b)
	if da != db {
		return da < db
	}
	ia := ba == "index" || ba == "README"
	ib := bb == "index" || bb == "README"
	if ia != ib {
		return ia
	}
	return ba < bb
}

// splitRel splits a slash-separated rel path into its directory and base name.
func splitRel(rel string) (dir, base string) {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[:i], rel[i+1:]
	}
	return "", rel
}
