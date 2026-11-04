package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// pureSite builds a site that uses every section and every kind of page the
// generator knows: tracks with relations, comments and replies, mentions, a
// release with shipped tracks, a blog, a portal, docs with code and a table of
// contents, and a source browser with a highlighted file, a diff and a log. The
// fixture text itself holds none of the characters the lint below forbids, so a
// hit is the generator's own markup.
func pureSite(t *testing.T) string {
	t.Helper()
	return generate(t, pureSiteRepo(t))
}

// pureSiteRepo is the repository behind pureSite, for tests that generate it more
// than once.
func pureSiteRepo(t *testing.T) *core.Repo {
	t.Helper()
	repo, dir := newSourceRepo(t)
	enableSections(t, repo, sectionPortal, sectionBlog, sectionDocs, sectionIssues, sectionReleases, sectionSource)
	a, err := repo.Create(core.NewTrackOpts{Title: "Build login", Description: "line one\nline two"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, _ := repo.Create(core.NewTrackOpts{Kind: "bug", Title: "Session leak"})
	done, _ := repo.Create(core.NewTrackOpts{Title: "Finished work"})
	if err := repo.Close(done, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := repo.Link(b, "depends-on", a); err != nil {
		t.Fatalf("Link: %v", err)
	}
	c1, err := repo.AddComment(a, "first comment", "")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if _, err := repo.AddComment(a, "a reply", c1); err != nil {
		t.Fatalf("AddComment reply: %v", err)
	}
	rel := mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", "## Notes\n\n```sh\nmake\n```\n")
	if err := repo.Link(rel, "ships", a); err != nil {
		t.Fatalf("Link ships: %v", err)
	}
	mkRelease(t, repo, "v2.0.0", "2025-06-01T00:00:00Z", "plain notes")

	slug, _ := repo.GetField(a, "slug")
	writeSiteFile(t, repo, "portal/front.md", "# Home\n\nWelcome.\n")
	writeSiteFile(t, repo, "portal/about.md", "# About\n\nText.\n")
	writeSiteFile(t, repo, "blog/post.md", "---\ntitle: Post\ndate: 2025-01-01\n---\n\nBody.\n\n```go\nfunc main() {}\n```\n")
	writeRepoFile(t, repo, "docs/guide.md", "# Guide\n\n"+numberedSectionsWithFence(9))
	writeRepoFile(t, repo, "docs/index.md", "# Docs\n\nStart here.\n")
	writeRepoText(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writeRepoText(t, dir, "notes.txt", "plain text\n")
	commitAll(t, dir, "2025-01-01T10:00:00Z", "first commit")
	writeRepoText(t, dir, "main.go", "package main\n\nfunc main() { println(1) }\n")
	commitAll(t, dir, "2025-02-02T11:30:00Z", "Fix it\n\ncloses #"+slug)
	return repo
}

// forbiddenInPages are things a page must not contain: presentation belongs to
// the theme (ADR-0048). Each is a pattern and why it is banned.
var forbiddenInPages = []struct {
	re   *regexp.Regexp
	why  string
	only string // when set, only pages under this prefix are checked
}{
	{regexp.MustCompile(`(?i)<style\b`), "inline stylesheet", ""},
	{regexp.MustCompile(`(?i)\sstyle\s*=`), "inline style attribute", ""},
	{regexp.MustCompile(`(?i)<script\b[^>]*>[^<]`), "inline script body", ""},
	{regexp.MustCompile(`(?i)<script\b(?:[^>]*\s)?(?:type\s*=\s*"?module)`), "module script (blocked from file://)", ""},
	{regexp.MustCompile(`(?i)\son[a-z]+\s*=`), "inline event handler", ""},
	{regexp.MustCompile(`(?i)</?(b|i|u|big|small|font|center|strike|tt)[\s>]`), "presentational element", ""},
	{regexp.MustCompile(`(?i)<hr\b`), "presentational rule", ""},
	{regexp.MustCompile(`&(middot|mdash|ndash|larr|rarr|minus|bull|raquo|laquo|nbsp);`), "decorative entity", ""},
	{regexp.MustCompile("[\u00b7\u2014\u2013\u2190\u2192\u2212\u2022]"), "decorative character", ""},
	{regexp.MustCompile(`class="[^"]*\bmuted\b`), "presentational class name", ""},
	{regexp.MustCompile(`ADR-\d`), "internal reference", ""},
}

func allPages(t *testing.T, out string) map[string]string {
	t.Helper()
	pages := map[string]string{}
	err := filepath.WalkDir(out, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return werr
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(out, p)
		pages[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return pages
}

func TestPagesCarryNoPresentation(t *testing.T) {
	pages := allPages(t, pureSite(t))
	if len(pages) < 20 {
		t.Fatalf("the fixture should exercise many pages, got %d", len(pages))
	}
	for rel, page := range pages {
		for _, f := range forbiddenInPages {
			if loc := f.re.FindStringIndex(page); loc != nil {
				end := min(loc[1]+40, len(page))
				t.Errorf("%s: %s: %q", rel, f.why, page[max(0, loc[0]-20):end])
			}
		}
	}
}

func TestPagesHaveLandmarksAndHooks(t *testing.T) {
	pages := allPages(t, pureSite(t))
	for rel, page := range pages {
		for _, want := range []string{"<header>", `<nav class="site-nav" aria-label="Site">`, `<main class="site-main">`, "<footer>", `<body data-section="`} {
			if !strings.Contains(page, want) {
				t.Errorf("%s: missing %s", rel, want)
			}
		}
		if n := strings.Count(page, "<main"); n != 1 {
			t.Errorf("%s: want exactly one main landmark, got %d", rel, n)
		}
		if n := strings.Count(page, "<h1"); n != 1 {
			t.Errorf("%s: want exactly one h1, got %d", rel, n)
		}
	}
	// the page's own nav entry is marked current
	if !strings.Contains(pages["issues.html"], `<a href="issues.html" aria-current="page">issues</a>`) {
		t.Errorf("the issues page should mark its own entry current")
	}
	// values a theme styles on are attributes, not text
	for rel, want := range map[string]string{
		"issues.html": `data-state="closed"`,
		"board.html":  `data-state="blocked"`,
	} {
		if !strings.Contains(pages[rel], want) {
			t.Errorf("%s: missing hook %s", rel, want)
		}
	}
	// dates are machine-readable
	if !strings.Contains(pages["blog/post.html"], `<time datetime="2025-01-01">2025-01-01</time>`) {
		t.Errorf("a post date should be a <time>:\n%s", pages["blog/post.html"])
	}
	if !strings.Contains(pages["source/log.html"], `<time datetime="2025-02-02T11:30:00Z">`) {
		t.Errorf("a commit date should be a <time> with an ISO datetime")
	}
}

// Every link to a stylesheet or script goes to a file under assets/, and every
// such file exists, so a page never points at a missing asset.
func TestEveryAssetLinkResolves(t *testing.T) {
	out := pureSite(t)
	ref := regexp.MustCompile(`(?:href|src)="([^"]*assets/[^"]+)"`)
	seen := 0
	for rel, page := range allPages(t, out) {
		for _, m := range ref.FindAllStringSubmatch(page, -1) {
			seen++
			target := filepath.Join(out, filepath.Dir(filepath.FromSlash(rel)), filepath.FromSlash(m[1]))
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s links %s, which was not written", rel, m[1])
			}
		}
	}
	if seen == 0 {
		t.Fatalf("no asset links found")
	}
}
