package core

import (
	"testing"
)

func TestAddStatusAndNext(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// add a status pointing at a built-in; must succeed
	if err := repo.AddStatus("blocked", []string{"in-progress", StatusDone}); err != nil {
		t.Fatalf("AddStatus: %v", err)
	}
	next, err := repo.NextStatuses("blocked")
	if err != nil {
		t.Fatalf("NextStatuses: %v", err)
	}
	if len(next) != 2 || next[0] != "in-progress" || next[1] != "done" {
		t.Errorf("next = %v, want [in-progress done]", next)
	}
	// it now counts as a valid status
	if ok, _ := repo.statusExists("blocked"); !ok {
		t.Error("blocked should exist after add")
	}
}

func TestAddStatusRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)

	// pure-hex name is refused (collides with id shape)
	if err := repo.AddStatus("deadbeef", nil); err == nil {
		t.Error("pure-hex status name should be refused")
	}
	// duplicate of a built-in is refused
	if err := repo.AddStatus(StatusNew, nil); err == nil {
		t.Error("adding an existing status should be refused")
	}
	// next target that does not exist is refused
	if err := repo.AddStatus("review", []string{"nonexistent"}); err == nil {
		t.Error("next target that is not a status should be refused")
	}
}

func TestSetNextValidatesTargets(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)

	// set-next on a built-in is allowed; target must exist
	if err := repo.SetNextStatuses(StatusNew, []string{StatusDone}); err != nil {
		t.Fatalf("SetNextStatuses(new): %v", err)
	}
	if err := repo.SetNextStatuses(StatusNew, []string{"ghost"}); err == nil {
		t.Error("set-next to a nonexistent status should be refused")
	}
	// empty next = unrestricted
	if err := repo.SetNextStatuses(StatusNew, nil); err != nil {
		t.Fatalf("SetNextStatuses(empty): %v", err)
	}
	if n, _ := repo.NextStatuses(StatusNew); len(n) != 0 {
		t.Errorf("empty next-set should be unrestricted, got %v", n)
	}
}

func TestRemoveStatusBuiltinRefused(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)
	repo.AddStatus("temp", nil)

	if err := repo.RemoveStatus("temp"); err != nil {
		t.Fatalf("RemoveStatus(temp): %v", err)
	}
	if ok, _ := repo.statusExists("temp"); ok {
		t.Error("temp should be gone")
	}
	// built-ins cannot be removed
	for _, b := range BuiltinStatuses {
		if err := repo.RemoveStatus(b); err == nil {
			t.Errorf("RemoveStatus(%q) should be refused (built-in)", b)
		}
	}
}
