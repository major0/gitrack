package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetObjectScalarList(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "Accessor Target"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// seed a comment file so a directory descent has something to list
	cdir := filepath.Join(repo.trackPath(id), "comments")
	if err := os.MkdirAll(cdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cdir, "c1.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatalf("write comment: %v", err)
	}

	// whole object
	v, err := repo.Get(id)
	if err != nil || v.Kind != "object" || v.Object.Title != "Accessor Target" {
		t.Fatalf("Get(object) = %+v, %v", v, err)
	}
	// scalar field
	v, err = repo.Get(id + ".status")
	if err != nil || v.Kind != "scalar" || v.Scalar != StatusNew {
		t.Fatalf("Get(.status) = %+v, %v; want scalar %q", v, err, StatusNew)
	}
	// directory -> list of BARE ids (storage suffix .md stripped, ADR-0022)
	v, err = repo.Get(id + ".comments")
	if err != nil || v.Kind != "list" || len(v.List) != 1 || v.List[0] != "c1" {
		t.Fatalf("Get(.comments) = %+v, %v; want [c1]", v, err)
	}
	// nested file by bare id: comments.c1 -> comments/c1.md
	v, err = repo.Get(id + ".comments.c1")
	if err != nil || v.Kind != "scalar" || v.Scalar != "hi" {
		t.Fatalf("Get(.comments.c1) = %+v, %v", v, err)
	}
	// a non-declared-suffix entry must NOT be matched by bare segment
	if err := os.WriteFile(filepath.Join(cdir, "note.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if _, err := repo.Get(id + ".comments.note"); err == nil {
		t.Error("segment 'note' must not match note.txt (.txt is not comments' suffix)")
	}
	// missing path
	if _, err := repo.Get(id + ".nope"); err == nil {
		t.Error("Get(.nope) should error")
	}
}

func TestSetFreeFormAndGovernedRejection(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// free-form write succeeds
	if err := repo.Set(id+".title", "Renamed"); err != nil {
		t.Fatalf("Set(.title): %v", err)
	}
	if got, _ := repo.GetField(id, "title"); got != "Renamed" {
		t.Errorf("title = %q, want Renamed", got)
	}
	// governed fields are refused
	for _, key := range []string{"state", "status"} {
		if err := repo.Set(id+"."+key, "whatever"); err == nil {
			t.Errorf("Set(.%s) should be refused", key)
		}
	}
	// nested path is not a valid set target
	if err := repo.Set(id+".comments.c1", "x"); err == nil {
		t.Error("Set of a nested path should error")
	}
}
