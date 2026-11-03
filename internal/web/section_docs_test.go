package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// writeRepoFile writes a file relative to the repo root (for docs sources that
// live outside .gitrack).
func writeRepoFile(t *testing.T, repo *core.Repo, rel, content string) {
	t.Helper()
	p := filepath.Join(repo.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func readOut(t *testing.T, out, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("missing %s: %v", rel, err)
	}
	return string(b)
}

func TestDocsAutoLayoutFromTree(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, _ := repo.Create(core.NewTrackOpts{Title: "Doc work"})
	slug, _ := repo.GetField(id, "slug")

	enableSections(t, repo, sectionIssues, sectionDocs)
	writeRepoFile(t, repo, "docs/intro.md", "# Introduction\n\nStart here. See #"+slug+".\n")
	writeRepoFile(t, repo, "docs/guide/setup.md", "# Setup\n\nInstall it.\n")

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// each doc renders under docs/ preserving its subpath
	intro := readOut(t, out, "docs/intro.html")
	if !strings.Contains(intro, "Introduction") || !strings.Contains(intro, "Start here") {
		t.Errorf("intro doc wrong:\n%s", intro)
	}
	// #ref in a doc body autolinks (relative from docs/intro.html => ../)
	if !strings.Contains(intro, `href="../track/`+id+`.html"`) {
		t.Errorf("doc #ref should autolink relative to the page:\n%s", intro)
	}
	setup := readOut(t, out, "docs/guide/setup.html")
	if !strings.Contains(setup, "Setup") {
		t.Errorf("nested doc missing:\n%s", setup)
	}
	// nested page links back to the site root with the right depth (../../)
	if !strings.Contains(setup, `href="../../index.html"`) {
		t.Errorf("nested doc nav should use ../../ to reach root:\n%s", setup)
	}
	// landing exists; auto sidebar shows the "guide" group and the intro entry
	landing := readOut(t, out, "docs/index.html")
	if !strings.Contains(landing, "docs-nav") {
		t.Errorf("landing should carry the sidebar:\n%s", landing)
	}
	if !strings.Contains(landing, "guide") {
		t.Errorf("auto sidebar should show the guide group:\n%s", landing)
	}
	// top-bar nav has a docs entry
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(index), `href="docs/index.html">docs`) {
		t.Errorf("top nav should have a docs entry:\n%s", index)
	}
}

func TestDocsConfiguredSource(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionIssues, sectionDocs)
	writeSiteFile(t, repo, "docs/source", "documentation\n")
	writeRepoFile(t, repo, "documentation/overview.md", "# Overview\n\nhi\n")

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// output prefix stays docs/ even though the source dir is "documentation"
	ov := readOut(t, out, "docs/overview.html")
	if !strings.Contains(ov, "Overview") {
		t.Errorf("configured-source doc missing:\n%s", ov)
	}
}

func TestDocsAuthoredLayout(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionIssues, sectionDocs)
	writeRepoFile(t, repo, "docs/intro.md", "# Introduction\n\nhi\n")
	writeRepoFile(t, repo, "docs/advanced.md", "# Advanced\n\ndeep\n")
	writeRepoFile(t, repo, "docs/orphan.md", "# Orphan\n\nnot listed\n")
	// layout: a group with intro under it, an explicit label, and a missing doc
	layout := "Getting Started\n  intro.md: First Steps\n  nope.md: Dead Entry\nadvanced.md\n"
	writeSiteFile(t, repo, "docs/layout", layout)

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	landing := readOut(t, out, "docs/index.html")

	// explicit label used for the page entry
	if !strings.Contains(landing, "First Steps") {
		t.Errorf("layout label should be used:\n%s", landing)
	}
	// group heading rendered as plain text, not a link
	if !strings.Contains(landing, `docs-group">Getting Started`) {
		t.Errorf("group heading should render as a plain label:\n%s", landing)
	}
	// missing doc entry degrades to a plain label (no link, no error)
	if strings.Contains(landing, `href="docs/nope`) {
		t.Errorf("missing-doc entry must not be a link:\n%s", landing)
	}
	if !strings.Contains(landing, "Dead Entry") {
		t.Errorf("missing-doc entry should still show its label:\n%s", landing)
	}
	// orphan doc (not in layout) still renders and appears under "Other"
	if _, err := os.Stat(filepath.Join(out, "docs", "orphan.html")); err != nil {
		t.Errorf("unlisted doc should still render: %v", err)
	}
	if !strings.Contains(landing, "Other") || !strings.Contains(landing, "Orphan") {
		t.Errorf("unlisted doc should be surfaced under Other:\n%s", landing)
	}
}

func TestDocsAbsentSourceEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionIssues, sectionDocs)
	// no docs/ dir
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "docs")); !os.IsNotExist(err) {
		t.Errorf("no docs/ output expected with no source (err=%v)", err)
	}
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if strings.Contains(string(index), "docs/index.html") {
		t.Errorf("no docs nav entry expected with no source:\n%s", index)
	}
}
