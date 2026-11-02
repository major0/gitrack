package core

import (
	"testing"
)

func TestNewUUIDv7ShapeAndVersion(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, err := NewUUIDv7()
		if err != nil {
			t.Fatalf("NewUUIDv7: %v", err)
		}
		if !IsUUID(id) {
			t.Fatalf("not a valid uuid shape: %q", id)
		}
		if id[14] != '7' {
			t.Errorf("want version 7 nibble, got %q in %s", id[14], id)
		}
		switch id[19] {
		case '8', '9', 'a', 'b':
		default:
			t.Errorf("want RFC9562 variant nibble, got %q in %s", id[19], id)
		}
		if seen[id] {
			t.Fatalf("duplicate uuid %s", id)
		}
		seen[id] = true
	}
}

func TestSlugify(t *testing.T) {
	const id = "0192f1a2-7c3e-7abc-def0-112233445566"
	cases := []struct {
		title string
		want  string
	}{
		{"New Login Flow", "new-login-flow"},
		{"  Trailing & leading!! ", "trailing-leading"},
		{"session refresh", "session-refresh"},
		{"", "track-0192f1a2"},   // empty title -> id-derived, non-hex
		{"deadbeef", "deadbeef"}, // not pure hex only because... it IS hex
	}
	for _, c := range cases {
		got := Slugify(c.title, id)
		// "deadbeef" is pure hex, so Slugify must avoid it.
		if c.title == "deadbeef" {
			if isHex(got) {
				t.Errorf("Slugify(%q) returned pure-hex slug %q", c.title, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"0192f1a2-7c3e-7abc-def0-112233445566": true,
		"0192f1a2-7c3e-7abc-def0-11223344556":  false, // too short
		"0192f1a27c3e7abcdef0112233445566":     false, // no hyphens
		"":                                     false,
		"gggggggg-7c3e-7abc-def0-112233445566": false, // non-hex
	}
	for in, want := range cases {
		if got := IsUUID(in); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCreateLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "New Login Flow", Description: "body text"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	tr, err := repo.Load(id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tr.Kind != "task" {
		t.Errorf("default kind = %q, want task", tr.Kind)
	}
	if tr.Slug != "new-login-flow" {
		t.Errorf("slug = %q, want new-login-flow", tr.Slug)
	}
	if tr.State != StateOpen {
		t.Errorf("state = %q, want open", tr.State)
	}
	if tr.Status != StatusNew {
		t.Errorf("status = %q, want new", tr.Status)
	}
	if tr.Description != "body text" {
		t.Errorf("description = %q", tr.Description)
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "Unique Title Here"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got, err := repo.Resolve(id); err != nil || got != id {
		t.Errorf("Resolve(full) = %q, %v; want %q", got, err, id)
	}
	if got, err := repo.Resolve("#unique-title-here"); err != nil || got != id {
		t.Errorf("Resolve(slug) = %q, %v; want %q", got, err, id)
	}
	if got, err := repo.Resolve("nonexistent-slug"); err == nil {
		t.Errorf("Resolve(bad slug) = %q, want error", got)
	}
}

func TestIsClosed(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if closed, _ := repo.IsClosed(id); closed {
		t.Error("new track should be open")
	}
	// ADR-0019: closed means exactly state == "closed".
	if err := repo.Close(id, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if closed, _ := repo.IsClosed(id); !closed {
		t.Error("closed state should be closed")
	}
}

func TestCreateSetsOpenAndNew(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	tr, err := repo.Load(id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tr.State != StateOpen || tr.Status != StatusNew {
		t.Errorf("new track = state %q status %q, want open/new", tr.State, tr.Status)
	}
}

func TestCloseReopenCycle(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, _ := repo.Create(NewTrackOpts{Title: "x"})
	if err := repo.Close(id, "wontfix"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	tr, _ := repo.Load(id)
	if tr.State != StateClosed || tr.Status != "wontfix" {
		t.Errorf("after close = %q/%q, want closed/wontfix", tr.State, tr.Status)
	}
	if err := repo.Reopen(id); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	tr, _ = repo.Load(id)
	if tr.State != StateOpen || tr.Status != StatusReopened {
		t.Errorf("after reopen = %q/%q, want open/reopened", tr.State, tr.Status)
	}
}

func TestValidateStatusTransition(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Seeded workflow: new -> {triaged,in-progress,done}.
	if err := repo.ValidateStatusTransition("new", "triaged"); err != nil {
		t.Errorf("new->triaged should be allowed: %v", err)
	}
	if err := repo.ValidateStatusTransition("new", "in-review"); err == nil {
		t.Error("new->in-review should be rejected (not in new's next set)")
	}
	// Built-in statuses are always allowed targets.
	if err := repo.ValidateStatusTransition("in-progress", StatusDone); err != nil {
		t.Errorf("any->done should be allowed: %v", err)
	}
	if err := repo.ValidateStatusTransition("done", StatusReopened); err != nil {
		t.Errorf("done->reopened should be allowed: %v", err)
	}
}
