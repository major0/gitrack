package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// setup creates an initialized repo with two tracks where B depends-on A.
func setup(t *testing.T) (*core.Repo, string, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	a, err := repo.Create(core.NewTrackOpts{Title: "track a"})
	if err != nil {
		t.Fatalf("Create a: %v", err)
	}
	b, err := repo.Create(core.NewTrackOpts{Kind: "bug", Title: "track b"})
	if err != nil {
		t.Fatalf("Create b: %v", err)
	}
	// B depends-on A (forward link, authoritative).
	rel := filepath.Join(repo.Root, core.Dir, b, "depends-on")
	if err := os.MkdirAll(rel, 0o755); err != nil {
		t.Fatalf("mkdir depends-on: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rel, a), nil, 0o644); err != nil {
		t.Fatalf("write depends-on: %v", err)
	}
	return repo, a, b
}

func TestRebuildProducesExpectedEntries(t *testing.T) {
	repo, a, b := setup(t)
	if _, err := Rebuild(repo); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	root := filepath.Join(repo.Root, core.Dir, "cache")
	wantExist := []string{
		filepath.Join("state", "open", a), // ADR-0019: new tracks are open
		filepath.Join("state", "open", b),
		filepath.Join("status", "new", a), // ...with status new
		filepath.Join("status", "new", b),
		filepath.Join("type", "task", a),
		filepath.Join("type", "bug", b),
		filepath.Join("refs-in", a, "depends-on", b), // reverse of B depends-on A
	}
	for _, rel := range wantExist {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("expected cache entry missing: %s (%v)", rel, err)
		}
	}
}

func TestVerifyCleanAfterRebuild(t *testing.T) {
	repo, _, _ := setup(t)
	if _, err := Rebuild(repo); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	rep, err := Verify(repo)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !rep.Clean() {
		t.Errorf("expected clean, got missing=%v extra=%v dangling=%v",
			rep.Missing, rep.Extra, rep.Dangling)
	}
}

func TestVerifyDetectsDrift(t *testing.T) {
	repo, a, _ := setup(t)
	if _, err := Rebuild(repo); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	// Inject a stale entry (a in a bucket it does not belong to).
	stale := filepath.Join(repo.Root, core.Dir, "cache", "state", "closed", a)
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(stale, nil, 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}
	rep, err := Verify(repo)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if rep.Clean() {
		t.Error("expected drift to be detected")
	}
	if len(rep.Extra) != 1 {
		t.Errorf("expected 1 extra entry, got %v", rep.Extra)
	}
}

func TestGuardrailWarnings(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(core.NewTrackOpts{Title: "bypass me"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Bypass the CLI: hand-edit the track into states/statuses/kinds that the
	// config does not define (ADR-0026 — the tree is openly editable).
	td := func(field, val string) {
		if e := os.WriteFile(filepath.Join(repo.Root, core.Dir, id, field), []byte(val+"\n"), 0o644); e != nil {
			t.Fatalf("write %s: %v", field, e)
		}
	}
	td("status", "invented-status")
	td("state", "weird")
	td("kind", "not-a-kind")

	rep, err := Verify(repo)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(rep.Warnings) != 3 {
		t.Fatalf("want 3 guardrail warnings, got %d: %v", len(rep.Warnings), rep.Warnings)
	}
	if rep.Clean() {
		t.Error("Clean() must be false when guardrail warnings exist")
	}
	// fsck must NOT rewrite authored fields (ADR-0026): the bad values persist.
	if s, _ := repo.GetField(id, "status"); s != "invented-status" {
		t.Errorf("fsck altered authored status to %q", s)
	}
}

func TestDanglingForwardLink(t *testing.T) {
	repo, _, b := setup(t)
	// Add a depends-on to a nonexistent track.
	rel := filepath.Join(repo.Root, core.Dir, b, "depends-on")
	ghost := "deadbeef-0000-7000-8000-000000000000"
	if err := os.WriteFile(filepath.Join(rel, ghost), nil, 0o644); err != nil {
		t.Fatalf("write ghost: %v", err)
	}
	rep, err := Verify(repo)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(rep.Dangling) == 0 {
		t.Error("expected a dangling forward link to be reported")
	}
}
