package core

import "testing"

func entryByID(b *Board, id string) BoardEntry {
	for _, e := range b.Entries {
		if e.ID == id {
			return e
		}
	}
	return BoardEntry{}
}

func TestComputeBoardDirectionsAndFrontier(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)
	ids := mkTracks(t, repo, 3)
	epic, child, dep := ids[0], ids[1], ids[2]

	// child child-of epic; epic depends-on dep.
	if err := repo.Link(child, "child-of", epic); err != nil {
		t.Fatalf("child-of: %v", err)
	}
	if err := repo.Link(epic, "depends-on", dep); err != nil {
		t.Fatalf("depends-on: %v", err)
	}

	b, err := repo.ComputeBoard()
	if err != nil {
		t.Fatalf("ComputeBoard: %v", err)
	}

	// child is Ready (not blocked by its open parent, ADR-0005).
	if e := entryByID(b, child); !e.Ready() {
		t.Errorf("child should be Ready, openBlockers=%v", e.OpenBlockers)
	}
	// dep is Ready (nothing blocks it).
	if e := entryByID(b, dep); !e.Ready() {
		t.Error("dep should be Ready")
	}
	// epic is Blocked by both its open child and its open dependency.
	if e := entryByID(b, epic); e.Ready() || len(e.OpenBlockers) != 2 {
		t.Errorf("epic should be blocked by 2, got %v", e.OpenBlockers)
	}

	// edges: dep->epic (depends-on reversed) and child->epic (child-of as-is).
	wantEdges := map[string]bool{dep + "->" + epic: true, child + "->" + epic: true}
	if len(b.Edges) != 2 {
		t.Fatalf("want 2 edges, got %d: %v", len(b.Edges), b.Edges)
	}
	for _, e := range b.Edges {
		if !wantEdges[e.From+"->"+e.To] {
			t.Errorf("unexpected edge %s -> %s", e.From, e.To)
		}
	}

	// close child and dep -> epic becomes Ready.
	if err := repo.Close(child, ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(dep, ""); err != nil {
		t.Fatal(err)
	}
	b, _ = repo.ComputeBoard()
	if e := entryByID(b, epic); !e.Ready() {
		t.Errorf("epic should be Ready after child and dep close, openBlockers=%v", e.OpenBlockers)
	}
	// closed tracks are not Open.
	if e := entryByID(b, child); e.Open {
		t.Error("closed child should not be Open")
	}
}
