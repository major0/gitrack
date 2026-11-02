package web

import (
	"html/template"
	"sort"

	"github.com/major0/gitrack/internal/core"
)

// commentNode is a comment in the rendered thread tree: its metadata, the body
// with #refs autolinked, and its replies (ADR-0023 reply-to threading).
type commentNode struct {
	ID       string
	Author   string
	Created  string
	BodyHTML template.HTML
	Replies  []*commentNode
}

// buildThread turns a flat comment list into a reply-to tree. Top-level comments
// (no reply-to, or a reply-to that does not resolve to a sibling) are roots;
// replies nest under their parent. Bodies are autolinked (ADR-0030) using root
// as the page's path-to-site-root. Ordering is by id (UUIDv7 -> chronological).
func buildThread(repo *core.Repo, root string, comments []core.CommentMeta) []*commentNode {
	nodes := make(map[string]*commentNode, len(comments))
	for _, c := range comments {
		nodes[c.ID] = &commentNode{
			ID:       c.ID,
			Author:   c.Author,
			Created:  c.Created,
			BodyHTML: autolink(repo, root, c.Body),
		}
	}
	var roots []*commentNode
	// attach each node to its parent (if the parent exists), else it is a root
	for _, c := range comments {
		n := nodes[c.ID]
		if parent, ok := nodes[c.ReplyTo]; c.ReplyTo != "" && ok {
			parent.Replies = append(parent.Replies, n)
		} else {
			roots = append(roots, n)
		}
	}
	sortNodes(roots)
	return roots
}

// sortNodes orders a slice of nodes and their replies by id (chronological for
// UUIDv7), recursively, for deterministic output.
func sortNodes(ns []*commentNode) {
	sort.Slice(ns, func(i, j int) bool { return ns[i].ID < ns[j].ID })
	for _, n := range ns {
		sortNodes(n.Replies)
	}
}
