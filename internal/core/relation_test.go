package core

import "testing"

// mk creates n tracks and returns their ids.
func mkTracks(t *testing.T, repo *Repo, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		id, err := repo.Create(NewTrackOpts{Title: "t"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		ids[i] = id
	}
	return ids
}

func TestLinkAndUnlink(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)
	ids := mkTracks(t, repo, 2)
	a, b := ids[0], ids[1]

	if err := repo.Link(a, "depends-on", b); err != nil {
		t.Fatalf("Link: %v", err)
	}
	tr, _ := repo.Load(a)
	if got := tr.Relations["depends-on"]; len(got) != 1 || got[0] != b {
		t.Errorf("depends-on = %v, want [%s]", got, b)
	}
	// unknown relation refused
	if err := repo.Link(a, "bogus", b); err == nil {
		t.Error("unknown relation should be refused")
	}
	// self-link refused
	if err := repo.Link(a, "related", a); err == nil {
		t.Error("self relationship should be refused")
	}
	// nonexistent target refused
	if err := repo.Link(a, "depends-on", "ffffffff-0000-7000-8000-000000000000"); err == nil {
		t.Error("nonexistent target should be refused")
	}
	// unlink removes it
	if err := repo.Unlink(a, "depends-on", b); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if err := repo.Unlink(a, "depends-on", b); err == nil {
		t.Error("unlinking a missing edge should error")
	}
}

func TestChildOfSingleParent(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)
	ids := mkTracks(t, repo, 3)
	child, p1, p2 := ids[0], ids[1], ids[2]

	if err := repo.Link(child, "child-of", p1); err != nil {
		t.Fatalf("first child-of: %v", err)
	}
	// a second, different parent is refused (ADR-0004 single-parent convention)
	if err := repo.Link(child, "child-of", p2); err == nil {
		t.Error("a second parent should be refused")
	}
	// re-linking the same parent is idempotent, not an error
	if err := repo.Link(child, "child-of", p1); err != nil {
		t.Errorf("re-linking same parent should be ok: %v", err)
	}
}

func TestBlockersDirection(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)
	ids := mkTracks(t, repo, 3)
	epic, childA, dep := ids[0], ids[1], ids[2]

	// childA child-of epic  => childA blocks epic (parent waits for child).
	if err := repo.Link(childA, "child-of", epic); err != nil {
		t.Fatalf("child-of: %v", err)
	}
	// epic depends-on dep    => dep blocks epic.
	if err := repo.Link(epic, "depends-on", dep); err != nil {
		t.Fatalf("depends-on: %v", err)
	}

	blk, err := repo.Blockers(epic)
	if err != nil {
		t.Fatalf("Blockers: %v", err)
	}
	// epic is blocked by its open child AND its open dependency.
	if len(blk) != 2 {
		t.Fatalf("epic blockers = %v, want childA and dep", blk)
	}
	if ub, _ := repo.IsUnblocked(epic); ub {
		t.Error("epic should be blocked (child and dep are open)")
	}
	// the child itself is NOT blocked by its parent (direction, ADR-0005).
	if cb, _ := repo.Blockers(childA); len(cb) != 0 {
		t.Errorf("child should have no blockers, got %v", cb)
	}
	if ub, _ := repo.IsUnblocked(childA); !ub {
		t.Error("child should be unblocked regardless of parent state")
	}

	// close the child and the dependency; epic becomes unblocked.
	if err := repo.Close(childA, ""); err != nil {
		t.Fatalf("close child: %v", err)
	}
	if err := repo.Close(dep, ""); err != nil {
		t.Fatalf("close dep: %v", err)
	}
	if ub, _ := repo.IsUnblocked(epic); !ub {
		t.Error("epic should be unblocked once child and dep are closed")
	}
}
