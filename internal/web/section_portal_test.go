package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// writeSiteFile writes a content file under .gitrack/config/site/<rel>.
func writeSiteFile(t *testing.T, repo *core.Repo, rel, content string) {
	t.Helper()
	p := filepath.Join(repo.Root, core.Dir, siteConfigDir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func enableSections(t *testing.T, repo *core.Repo, names ...string) {
	t.Helper()
	writeSiteFile(t, repo, "sections", strings.Join(names, "\n")+"\n")
}

func TestPortalSectionRendersPagesAndHome(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, _ := repo.Create(core.NewTrackOpts{Title: "Login Flow"})
	slug, _ := repo.GetField(id, "slug")

	enableSections(t, repo, sectionPortal, sectionIssues)
	writeSiteFile(t, repo, "portal/front.md", "# Welcome\n\nHome page. See #"+slug+" for work.\n")
	writeSiteFile(t, repo, "portal/about.md", "---\ntitle: About Us\n---\n\nWe build things.\n")

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

	// front page owns index.html (portal is first/home section)
	index := read("index.html")
	if !strings.Contains(index, "Welcome") || !strings.Contains(index, "Home page") {
		t.Errorf("index.html should be the portal front page:\n%s", index)
	}
	// #ref in the front page autolinks (ADR-0038)
	if !strings.Contains(index, `href="track/`+id+`.html"`) {
		t.Errorf("front page #ref should autolink:\n%s", index)
	}
	// about page at portal/about.html, title from front matter
	about := read("portal/about.html")
	if !strings.Contains(about, "We build things") || !strings.Contains(about, "About Us") {
		t.Errorf("about page wrong:\n%s", about)
	}
	// issues track index relocated to tracks.html (portal owns home)
	if _, err := os.Stat(filepath.Join(out, "tracks.html")); err != nil {
		t.Errorf("track index should move to tracks.html when portal is home: %v", err)
	}
	// nav shows home, the About page, and the relocated tracks link
	if !strings.Contains(index, `href="index.html">home`) {
		t.Errorf("nav missing home entry:\n%s", index)
	}
	if !strings.Contains(index, `>About Us</a>`) {
		t.Errorf("nav missing About page entry:\n%s", index)
	}
	if !strings.Contains(index, `href="tracks.html">tracks`) {
		t.Errorf("nav should point tracks to tracks.html:\n%s", index)
	}
}

func TestPortalAbsentContentEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// portal enabled but no portal/ dir: portal cannot provide a home (no
	// front.md), so home falls through to issues, which writes the track list
	// to index.html as usual (ADR-0039).
	enableSections(t, repo, sectionPortal, sectionIssues)
	repo.Create(core.NewTrackOpts{Title: "A track"}) //nolint:errcheck // seed
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("index.html should exist (issues falls back to home): %v", err)
	}
	if !strings.Contains(string(index), "Tracks") {
		t.Errorf("index.html should be the track list when portal has no front page:\n%s", index)
	}
	// and tracks.html should NOT exist, since issues is home
	if _, err := os.Stat(filepath.Join(out, "tracks.html")); !os.IsNotExist(err) {
		t.Errorf("tracks.html should not exist when issues is home (err=%v)", err)
	}
}
