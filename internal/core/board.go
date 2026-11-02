package core

import (
	"fmt"
	"sort"
	"strings"
)

// BoardEntry is one track's place on the board: its id/title/status, whether it
// is open, and the ids of its OPEN blockers (empty means ready).
type BoardEntry struct {
	ID           string
	Title        string
	Status       string
	Open         bool
	OpenBlockers []string
}

// Edge is a blocking-graph edge: From blocks To (an arrow points at the waiter),
// per ADR-0005.
type Edge struct {
	From string
	To   string
}

// Board is the computed board: every track classified, plus the blocking edges.
// It is built in a single O(N + edges) pass (ADR-0027) — no per-track rescans.
type Board struct {
	Entries []BoardEntry
	Edges   []Edge
}

// ComputeBoard loads every track once, builds the blocking graph, and classifies
// each track. Blocking directions (ADR-0005):
//   - A depends-on B  => edge B -> A  (A waits for B)
//   - C child-of A    => edge C -> A  (the parent A waits for the child C)
func (r *Repo) ComputeBoard() (*Board, error) {
	ids, err := r.ListIDs()
	if err != nil {
		return nil, err
	}
	// Single load of each track.
	tracks := make(map[string]*Track, len(ids))
	for _, id := range ids {
		t, err := r.Load(id)
		if err != nil {
			return nil, err
		}
		tracks[id] = t
	}

	// Closed means exactly state == "closed" (ADR-0019).
	isClosed := func(id string) bool {
		t, ok := tracks[id]
		return ok && t.State == StateClosed
	}

	// Build blocking edges once, and per-track blocker lists.
	var edges []Edge
	blockers := make(map[string][]string, len(ids))
	addBlocker := func(blocker, blocked string) {
		edges = append(edges, Edge{From: blocker, To: blocked})
		blockers[blocked] = append(blockers[blocked], blocker)
	}
	for id, t := range tracks {
		// A depends-on B => B blocks A
		for _, b := range t.Relations["depends-on"] {
			if _, ok := tracks[b]; ok {
				addBlocker(b, id)
			}
		}
		// id child-of parent => id blocks parent
		for _, parent := range t.Relations["child-of"] {
			if _, ok := tracks[parent]; ok {
				addBlocker(id, parent)
			}
		}
	}

	board := &Board{Edges: edges}
	for _, id := range ids {
		t := tracks[id]
		open := !isClosed(id)
		var openBlockers []string
		for _, b := range blockers[id] {
			if !isClosed(b) {
				openBlockers = append(openBlockers, b)
			}
		}
		sort.Strings(openBlockers)
		board.Entries = append(board.Entries, BoardEntry{
			ID:           id,
			Title:        t.Title,
			Status:       t.Status,
			Open:         open,
			OpenBlockers: openBlockers,
		})
	}
	// Entries follow ListIDs order (UUIDv7 -> chronological); edges sorted for
	// deterministic output.
	sort.Slice(board.Edges, func(i, j int) bool {
		if board.Edges[i].From != board.Edges[j].From {
			return board.Edges[i].From < board.Edges[j].From
		}
		return board.Edges[i].To < board.Edges[j].To
	})
	return board, nil
}

// Ready reports whether an entry is actionable now: open with no open blocker.
func (e BoardEntry) Ready() bool {
	return e.Open && len(e.OpenBlockers) == 0
}

// Mermaid renders the board's blocking digraph as a Mermaid "graph LR" body
// (without the surrounding code fence, so callers can embed it in HTML or wrap
// it in a Markdown fence). Edges point blocker -> blocked (ADR-0005); ready and
// closed nodes get classes. Shared by the CLI and the serve board page.
func (b *Board) Mermaid() string {
	var sb strings.Builder
	sb.WriteString("graph LR\n")
	for _, e := range b.Entries {
		node := e.ID[:8]
		label := node + ": " + e.Title
		switch {
		case !e.Open:
			sb.WriteString(fmt.Sprintf("  %s[%q]:::closed\n", node, label))
		case e.Ready():
			sb.WriteString(fmt.Sprintf("  %s[%q]:::ready\n", node, label))
		default:
			sb.WriteString(fmt.Sprintf("  %s[%q]\n", node, label))
		}
	}
	for _, edge := range b.Edges {
		sb.WriteString(fmt.Sprintf("  %s --> %s\n", edge.From[:8], edge.To[:8]))
	}
	sb.WriteString("  classDef ready fill:#cfc,stroke:#2a2;\n")
	sb.WriteString("  classDef closed fill:#eee,stroke:#999,color:#999;\n")
	return sb.String()
}

// DOT renders the board's blocking digraph as a Graphviz DOT document.
func (b *Board) DOT() string {
	var sb strings.Builder
	sb.WriteString("digraph gitrack {\n  rankdir=LR;\n  node [shape=box];\n")
	for _, e := range b.Entries {
		node := e.ID[:8]
		label := dotEscape(node + ": " + e.Title)
		switch {
		case !e.Open:
			sb.WriteString(fmt.Sprintf("  %q [label=%q, style=filled, fillcolor=\"#eeeeee\", fontcolor=\"#999999\"];\n", node, label))
		case e.Ready():
			sb.WriteString(fmt.Sprintf("  %q [label=%q, style=filled, fillcolor=\"#ccffcc\"];\n", node, label))
		default:
			sb.WriteString(fmt.Sprintf("  %q [label=%q];\n", node, label))
		}
	}
	for _, edge := range b.Edges {
		sb.WriteString(fmt.Sprintf("  %q -> %q;\n", edge.From[:8], edge.To[:8]))
	}
	sb.WriteString("}\n")
	return sb.String()
}

// dotEscape escapes backslashes and quotes for a DOT label.
func dotEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}
