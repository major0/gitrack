package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestBlogSectionRendersPostsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionBlog, sectionIssues)
	writeSiteFile(t, repo, "blog/older.md", "---\ntitle: Older Post\ndate: 2026-01-01\n---\n\nFirst news.\n")
	writeSiteFile(t, repo, "blog/newer.md", "---\ntitle: Newer Post\ndate: 2026-06-01\n---\n\nFresh news.\n")

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(out, rel))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		return string(b)
	}

	// each post renders to blog/<slug>.html with its title and date
	newer := read("blog/newer.html")
	if !strings.Contains(newer, "Newer Post") || !strings.Contains(newer, "Fresh news") {
		t.Errorf("newer post wrong:\n%s", newer)
	}
	if !strings.Contains(newer, "2026-06-01") {
		t.Errorf("post should show its date:\n%s", newer)
	}

	// the index lists newest-first: Newer Post appears before Older Post
	idx := read("blog/index.html")
	ni := strings.Index(idx, "Newer Post")
	oi := strings.Index(idx, "Older Post")
	if ni < 0 || oi < 0 {
		t.Fatalf("index missing a post:\n%s", idx)
	}
	if ni > oi {
		t.Errorf("index should list newest-first (Newer before Older):\n%s", idx)
	}
}

func TestBlogUndatedSortsLastAndBareFileWorks(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionBlog, sectionIssues)
	// a dated post and a bare (no front matter, title from H1, undated) post
	writeSiteFile(t, repo, "blog/dated.md", "---\ndate: 2026-03-03\ntitle: Dated\n---\n\nbody\n")
	writeSiteFile(t, repo, "blog/bare.md", "# Bare Title\n\njust markdown\n")

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	idx, _ := os.ReadFile(filepath.Join(out, "blog", "index.html"))
	s := string(idx)
	// bare post title falls back to its H1
	if !strings.Contains(s, "Bare Title") {
		t.Errorf("bare post title should fall back to H1:\n%s", s)
	}
	// dated sorts before undated
	di := strings.Index(s, "Dated")
	bi := strings.Index(s, "Bare Title")
	if di < 0 || bi < 0 || di > bi {
		t.Errorf("dated post should sort before undated:\n%s", s)
	}
	// the bare post page still renders
	if _, err := os.Stat(filepath.Join(out, "blog", "bare.html")); err != nil {
		t.Errorf("bare post page should render: %v", err)
	}
}

func TestBlogAbsentContentEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionBlog, sectionIssues)
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "blog")); !os.IsNotExist(err) {
		t.Errorf("no blog/ dir expected when there are no posts (err=%v)", err)
	}
	// and the issues index should not carry a blog nav entry
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if strings.Contains(string(index), "blog/index.html") {
		t.Errorf("no blog nav entry expected when blog is empty:\n%s", index)
	}
}
