package web

import (
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestGithubSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Getting started", "getting-started"},
		{"Hello, World!", "hello-world"},
		{"A - B", "a---b"},
		{"C++ & Go", "c--go"},
		{"Version 2.0 (beta)", "version-20-beta"},
		{"  padded  ", "padded"},
		{"Tab\there", "tabhere"},
		// where GitHub and goldmark's own generator disagree
		{"snake_case names", "snake_case-names"},
		{"Café au lait", "café-au-lait"},
		{"ÀÉ", "àé"},
		{"日本語 の 見出し", "日本語-の-見出し"},
		// nothing slug-able
		{"!!!", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := githubSlug(c.in); got != c.want {
			t.Errorf("githubSlug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSluggerUniqueness(t *testing.T) {
	s := newSlugger()
	var got []string
	for _, h := range []string{"Intro", "Intro", "Intro", "Intro 1", "Intro", "!!!", "???"} {
		got = append(got, s.next(h))
	}
	// "Intro 1" slugs to intro-1, which the second heading already took, so it
	// is itself suffixed; the next plain "Intro" continues its own count.
	want := []string{"intro", "intro-1", "intro-2", "intro-1-1", "intro-3", "section", "section-1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

// headingIDs returns the id of every heading in rendered HTML, in order.
func headingIDs(html string) []string {
	var ids []string
	for _, m := range regexp.MustCompile(`<h[1-6] id="([^"]*)"`).FindAllStringSubmatch(html, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

func render(t *testing.T, src string) string {
	t.Helper()
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	out, err := renderMarkdown(repo, "", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	return string(out)
}

func TestHeadingsGetIdsAndAnAccessiblePermalink(t *testing.T) {
	html := render(t, "# Getting started\n\n## The `git track web` command\n")

	if got := headingIDs(html); strings.Join(got, ",") != "getting-started,the-git-track-web-command" {
		t.Errorf("ids = %v\n%s", got, html)
	}
	// the permalink sits inside its heading, links to its id, and has a name
	h1 := regexp.MustCompile(`<h1 id="getting-started">Getting started<a [^>]*>Permalink</a></h1>`)
	if !h1.MatchString(html) {
		t.Errorf("h1 should contain its text then a permalink:\n%s", html)
	}
	for _, want := range []string{`href="#getting-started"`, `class="anchor"`, `title="Permalink to this section"`} {
		if !strings.Contains(html, want) {
			t.Errorf("permalink missing %s:\n%s", want, html)
		}
	}
	// the id comes from the visible text, not the backticks around the code
	if !strings.Contains(html, `<h2 id="the-git-track-web-command">The <code>git track web</code> command`) {
		t.Errorf("inline code in a heading should be kept and slugged by its text:\n%s", html)
	}
}

func TestHeadingIdsMatchGithubWhereGoldmarkDoesNot(t *testing.T) {
	html := render(t, "## snake_case names\n\n## Café au lait\n\n## 日本語 の 見出し\n")
	want := "snake_case-names,café-au-lait,日本語-の-見出し"
	if got := strings.Join(headingIDs(html), ","); got != want {
		t.Errorf("ids = %q, want %q", got, want)
	}
	// a non-ASCII id is linked in a form browsers resolve: the href, once
	// percent-decoded, equals the id
	for _, m := range regexp.MustCompile(`<a [^>]*href="#([^"]*)"[^>]*class="anchor"|<a [^>]*class="anchor"[^>]*href="#([^"]*)"`).FindAllStringSubmatch(html, -1) {
		frag := m[1] + m[2]
		dec, err := url.PathUnescape(frag)
		if err != nil || !strings.Contains(want, dec) {
			t.Errorf("permalink fragment %q decodes to %q, which is not one of the ids", frag, dec)
		}
	}
}

func TestDuplicateHeadingsAreNumberedPerDocument(t *testing.T) {
	src := "## Intro\n\ntext\n\n## Intro\n\n## Intro\n"
	first := headingIDs(render(t, src))
	if strings.Join(first, ",") != "intro,intro-1,intro-2" {
		t.Errorf("ids = %v, want intro,intro-1,intro-2", first)
	}
	// a second document starts again from the beginning
	if second := headingIDs(render(t, src)); strings.Join(second, ",") != strings.Join(first, ",") {
		t.Errorf("a new document must restart numbering, got %v", second)
	}
}

func TestSetextAndNestedHeadingsGetIds(t *testing.T) {
	html := render(t, "Title\n=====\n\nSub\n---\n\n> ## Quoted\n\n- ### In a list\n")
	if got := strings.Join(headingIDs(html), ","); got != "title,sub,quoted,in-a-list" {
		t.Errorf("ids = %q\n%s", got, html)
	}
}

// maxAnchorDepth is the deepest <a> nesting in html; valid HTML never exceeds 1.
func maxAnchorDepth(html string) int {
	depth, max := 0, 0
	for _, tok := range regexp.MustCompile(`<a[ >]|</a>`).FindAllString(html, -1) {
		if tok == "</a>" {
			depth--
		} else {
			depth++
			if depth > max {
				max = depth
			}
		}
	}
	return max
}

func TestHeadingWithAutolinkedRefHasNoNestedAnchors(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, _ := repo.Create(core.NewTrackOpts{Title: "Login flow"})
	slug, _ := repo.GetField(id, "slug")

	out, err := renderMarkdown(repo, "", "## Fixes #"+slug+"\n")
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	html := string(out)
	if got := strings.Join(headingIDs(html), ","); got != "fixes-"+slug {
		t.Errorf("the id should come from the text, got %q", got)
	}
	if !strings.Contains(html, `href="track/`+id+`.html"`) {
		t.Errorf("the #ref in the heading should still link:\n%s", html)
	}
	if d := maxAnchorDepth(html); d != 1 {
		t.Errorf("links must not nest (depth %d):\n%s", d, html)
	}
	if strings.Count(html, "<a ") != 2 {
		t.Errorf("expected the ref link and the permalink:\n%s", html)
	}
}

func TestHeadingIdsAreSafeForHostileText(t *testing.T) {
	html := render(t, "## a \"quoted\" <b>bold</b> & 'more' <script>alert(1)</script>\n")
	if strings.Contains(html, "<script>") {
		t.Errorf("raw HTML must stay inert:\n%s", html)
	}
	safe := regexp.MustCompile(`^[\p{L}\p{M}\p{N}_-]+$`)
	ids := headingIDs(html)
	if len(ids) != 1 || !safe.MatchString(ids[0]) {
		t.Errorf("an id may only hold letters, marks, numbers, '_' and '-', got %v\n%s", ids, html)
	}
}

func TestCrossDocumentFragmentLinksKeepTheirFragment(t *testing.T) {
	html := render(t, "See [the install steps](guide.md#install) and [below](#details).\n\n## Details\n")
	for _, want := range []string{`href="guide.html#install"`, `href="#details"`, `id="details"`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s:\n%s", want, html)
		}
	}
}

// TestPublishedDocsFragmentLinksResolve renders the real docs tree and checks
// that every link of the form page#fragment points at an id that exists, so a
// renamed heading cannot silently break a link.
func TestPublishedDocsFragmentLinksResolve(t *testing.T) {
	realDocs := filepath.Join("..", "..", "docs")
	if _, err := os.Stat(realDocs); err != nil {
		t.Skipf("docs tree not found: %v", err)
	}
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// copy the published docs (everything except the internal adr/ directory)
	err = filepath.WalkDir(realDocs, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			if d.Name() == "adr" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(realDocs, p)
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		writeRepoText(t, repo.Root, "docs/"+filepath.ToSlash(rel), string(b))
		return nil
	})
	if err != nil {
		t.Fatalf("copying docs: %v", err)
	}
	enableSections(t, repo, sectionDocs)
	out := generate(t, repo)

	idRe := regexp.MustCompile(`\sid="([^"]+)"`)
	hrefRe := regexp.MustCompile(`href="([^"]*#[^"]*)"`)
	ids := map[string]map[string]bool{} // page (slash path under docs/) -> ids
	pages := map[string]string{}        // page -> html
	err = filepath.WalkDir(filepath.Join(out, "docs"), func(p string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return werr
		}
		rel, _ := filepath.Rel(filepath.Join(out, "docs"), p)
		rel = filepath.ToSlash(rel)
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		pages[rel] = string(b)
		ids[rel] = map[string]bool{}
		for _, m := range idRe.FindAllStringSubmatch(string(b), -1) {
			ids[rel][m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking output: %v", err)
	}

	checked := 0
	for page, html := range pages {
		for _, m := range hrefRe.FindAllStringSubmatch(html, -1) {
			target, frag, _ := strings.Cut(m[1], "#")
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			dest := page
			if target != "" {
				dest = path.Join(path.Dir(page), target)
			}
			set, ok := ids[dest]
			if !ok {
				continue // a link to a page outside docs/, not this test's concern
			}
			want, err := url.PathUnescape(frag)
			if err != nil {
				t.Errorf("%s: undecodable fragment %q", page, frag)
				continue
			}
			checked++
			if !set[want] {
				t.Errorf("%s links to %s#%s but that page has no such id", page, dest, want)
			}
		}
	}
	// permalinks alone guarantee checks happen; a docs tree with none would make
	// this test vacuous
	if checked == 0 {
		t.Errorf("no fragment links were checked")
	}
}
