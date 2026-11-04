package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestGenerateStaticSite(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	a, err := repo.Create(core.NewTrackOpts{Title: "Build login"})
	if err != nil {
		t.Fatalf("Create a: %v", err)
	}
	b, err := repo.Create(core.NewTrackOpts{Kind: "bug", Title: "Session leak"})
	if err != nil {
		t.Fatalf("Create b: %v", err)
	}
	if err := repo.Link(b, "depends-on", a); err != nil {
		t.Fatalf("Link: %v", err)
	}

	out := filepath.Join(t.TempDir(), "site")
	n, err := Generate(repo, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if n != 2 {
		t.Errorf("wrote %d track pages, want 2", n)
	}

	read := func(rel string) string {
		p := filepath.Join(out, rel)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		return string(data)
	}

	index := read("index.html")
	board := read("board.html")
	trackA := read("track/" + a + ".html")

	// index lists both tracks and links to track pages with RELATIVE paths
	if !strings.Contains(index, "Build login") || !strings.Contains(index, "Session leak") {
		t.Error("index missing a track title")
	}
	if !strings.Contains(index, `href="track/`+a+`.html"`) {
		t.Error("index should link to the track page with a relative path")
	}
	if strings.Contains(index, "http://") || strings.Contains(index, "https://") {
		// the index (core content) must have NO network dependency (ADR-0029)
		t.Error("index must not reference any network URL")
	}

	// presentation is a relative stylesheet file, never inline
	if !strings.Contains(index, `<link rel="stylesheet" href="assets/theme/theme.css">`) {
		t.Error("index should link the theme stylesheet by a relative path")
	}
	if strings.Contains(index, "<style") || strings.Contains(index, "style=") {
		t.Error("index must carry no inline CSS")
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "theme", "theme.css")); err != nil {
		t.Errorf("the theme stylesheet must be written: %v", err)
	}

	// board: Ready/Blocked lists are present as the offline-complete fallback
	if !strings.Contains(board, "Ready") || !strings.Contains(board, "Blocked") {
		t.Error("board missing Ready/Blocked sections")
	}
	// session depends-on login => session is Blocked, login is Ready
	if !strings.Contains(board, "Session leak") {
		t.Error("board should show the blocked track")
	}
	// the mermaid graph is present but is a PROGRESSIVE enhancement; the only
	// network reference in the whole site is that one enhancement script.
	if !strings.Contains(board, "graph LR") {
		t.Error("board should embed the mermaid source")
	}

	// track page uses a relative up-link and shows the relationship
	if !strings.Contains(trackA, `href="../index.html"`) {
		t.Error("track page should link back to index with a relative path")
	}
}

func TestGenerateIssuesView(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	openID, _ := repo.Create(core.NewTrackOpts{Title: "Still open"})
	doneID, _ := repo.Create(core.NewTrackOpts{Title: "Already done"})
	if err := repo.Close(doneID, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	issues, err := os.ReadFile(filepath.Join(out, "issues.html"))
	if err != nil {
		t.Fatalf("no issues.html: %v", err)
	}
	s := string(issues)
	// open section has the open track; closed section has the done one
	if !strings.Contains(s, `Open <span class="count">1</span>`) {
		t.Errorf("expected an Open count of 1; got page:\n%s", s)
	}
	if !strings.Contains(s, `Closed <span class="count">1</span>`) {
		t.Errorf("expected a Closed count of 1")
	}
	if !strings.Contains(s, "Still open") || !strings.Contains(s, "Already done") {
		t.Error("issues page missing a track title")
	}
	// links are relative to track pages
	if !strings.Contains(s, `href="track/`+openID+`.html"`) {
		t.Error("issues page should link to the track page relatively")
	}
	// nav links to issues from the index
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(index), `href="issues.html"`) {
		t.Error("index nav should link to issues")
	}
}

func TestGenerateOverwrites(t *testing.T) {
	dir := t.TempDir()
	repo, _ := core.Init(dir)
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	// a stale file from a prior run under track/ should be regenerated cleanly;
	// generating again must not error (idempotent).
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Errorf("index.html should exist after regenerate: %v", err)
	}
}
