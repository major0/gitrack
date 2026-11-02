package core

import "sort"

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
