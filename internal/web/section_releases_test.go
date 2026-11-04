package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// mkRelease creates a release track with a fixed created timestamp so its
// derived date is deterministic (no tag exists, so the date falls back to it).
func mkRelease(t *testing.T, repo *core.Repo, title, created, notes string) string {
	t.Helper()
	id, err := repo.Create(core.NewTrackOpts{Kind: core.KindRelease, Title: title, Description: notes})
	if err != nil {
		t.Fatalf("Create release %s: %v", title, err)
	}
	if err := repo.SetField(id, "created", created); err != nil {
		t.Fatalf("SetField created: %v", err)
	}
	return id
}

func TestReleasesSectionRendersIndexAndPages(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	openIssue, _ := repo.Create(core.NewTrackOpts{Title: "Login flow"})
	doneIssue, _ := repo.Create(core.NewTrackOpts{Title: "Session fix"})
	if err := repo.Close(doneIssue, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
	slug, _ := repo.GetField(openIssue, "slug")

	older := mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", "First cut.")
	newer := mkRelease(t, repo, "v2.0.0", "2025-06-01T00:00:00Z",
		"## Highlights\n\nBig one. See #"+slug+".\n")
	if err := repo.Close(older, ""); err != nil {
		t.Fatalf("Close older: %v", err)
	}
	if err := repo.SetField(newer, "tag", "v2.0.0-rc1"); err != nil {
		t.Fatalf("SetField tag: %v", err)
	}
	if err := repo.Link(newer, "ships", openIssue); err != nil {
		t.Fatalf("Link: %v", err)
	}
	if err := repo.Link(newer, "ships", doneIssue); err != nil {
		t.Fatalf("Link: %v", err)
	}
	enableSections(t, repo, sectionIssues, sectionReleases)

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// index: newest first, with derived dates and released/upcoming wording
	idx := readOut(t, out, "releases/index.html")
	ni, oi := strings.Index(idx, "v2.0.0"), strings.Index(idx, "v1.0.0")
	if ni < 0 || oi < 0 || ni > oi {
		t.Errorf("index should list v2.0.0 before v1.0.0:\n%s", idx)
	}
	for _, want := range []string{"2025-06-01", "2025-01-01", "upcoming", "released", "2 shipped"} {
		if !strings.Contains(idx, want) {
			t.Errorf("index missing %q:\n%s", want, idx)
		}
	}

	// release page: notes rendered as Markdown, #ref autolinked, tag shown,
	// shipped tracks linked to their track pages
	page := readOut(t, out, "releases/v2-0-0.html")
	if !strings.Contains(page, "<h2") || !strings.Contains(page, "Highlights") {
		t.Errorf("notes should render as Markdown:\n%s", page)
	}
	if !strings.Contains(page, `href="../track/`+openIssue+`.html"`) {
		t.Errorf("shipped track should link to its track page:\n%s", page)
	}
	if !strings.Contains(page, "<code>v2.0.0-rc1</code>") {
		t.Errorf("explicit tag should be shown:\n%s", page)
	}
	if !strings.Contains(page, "Shipped (2)") {
		t.Errorf("shipped count missing:\n%s", page)
	}

	// top nav gains a releases entry
	home := readOut(t, out, "index.html")
	if !strings.Contains(home, `href="releases/index.html">releases`) {
		t.Errorf("nav should have a releases entry:\n%s", home)
	}

	// the shipped track's own page links back to the release that shipped it
	track := readOut(t, out, "track/"+openIssue+".html")
	if !strings.Contains(track, "shipped in:") || !strings.Contains(track, `href="../releases/v2-0-0.html"`) {
		t.Errorf("shipped track page should link back to its release:\n%s", track)
	}
}

func TestReleasesNoneEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	repo.Create(core.NewTrackOpts{Title: "Just an issue"}) //nolint:errcheck // seed
	enableSections(t, repo, sectionIssues, sectionReleases)

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "releases")); !os.IsNotExist(err) {
		t.Errorf("no releases/ expected without release tracks (err=%v)", err)
	}
	home := readOut(t, out, "index.html")
	if strings.Contains(home, "releases/index.html") {
		t.Errorf("no releases nav entry expected:\n%s", home)
	}
}

func TestReleasesWithoutIssuesSectionHasNoDeadLinks(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	issue, _ := repo.Create(core.NewTrackOpts{Title: "Login flow"})
	rel := mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", "notes")
	if err := repo.Link(rel, "ships", issue); err != nil {
		t.Fatalf("Link: %v", err)
	}
	enableSections(t, repo, sectionReleases) // issues section OFF

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	page := readOut(t, out, "releases/v1-0-0.html")
	if !strings.Contains(page, "Login flow") {
		t.Errorf("shipped track title should still be listed:\n%s", page)
	}
	if strings.Contains(page, `href="../track/`) {
		t.Errorf("track pages do not exist with issues off; no link expected:\n%s", page)
	}
}

func TestShippedInLinksToTrackPageWhenReleasesOff(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	issue, _ := repo.Create(core.NewTrackOpts{Title: "Login flow"})
	rel := mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", "notes")
	if err := repo.Link(rel, "ships", issue); err != nil {
		t.Fatalf("Link: %v", err)
	}
	enableSections(t, repo, sectionIssues) // releases section OFF

	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	track := readOut(t, out, "track/"+issue+".html")
	if !strings.Contains(track, `href="`+rel+`.html"`) {
		t.Errorf("without the releases section, shipped-in should link the release's track page:\n%s", track)
	}
	if strings.Contains(track, "../releases/") {
		t.Errorf("no releases/ pages exist; link must not point there:\n%s", track)
	}
}
