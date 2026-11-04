package web

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// renderDocWithTOC renders Markdown with a table of contents at tocMin.
func renderDocWithTOC(t *testing.T, src string, tocMin int) string {
	t.Helper()
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	res, err := renderMarkdownWith(markdownOpts{repo: repo, tocMin: tocMin}, src)
	if err != nil {
		t.Fatalf("renderMarkdownWith: %v", err)
	}
	return string(res.HTML)
}

// numberedSections builds a page with a title, an intro, and n h2 sections.
func numberedSections(n int) string {
	var b strings.Builder
	b.WriteString("# Title\n\nAn introduction.\n\n")
	for i := 1; i <= n; i++ {
		b.WriteString("## Section " + strconv.Itoa(i) + "\n\nSome text.\n\n")
	}
	return b.String()
}

var tocBlockRe = regexp.MustCompile(`(?s)<nav class="toc".*?</nav>`)

// tocBlock returns the table of contents in rendered HTML, or "" if there is none.
func tocBlock(html string) string { return tocBlockRe.FindString(html) }

// tocFragments returns the decoded fragment of every link in the TOC, in order.
func tocFragments(t *testing.T, html string) []string {
	t.Helper()
	var out []string
	for _, m := range regexp.MustCompile(`href="#([^"]*)"`).FindAllStringSubmatch(tocBlock(html), -1) {
		dec, err := url.PathUnescape(m[1])
		if err != nil {
			t.Fatalf("undecodable TOC fragment %q: %v", m[1], err)
		}
		out = append(out, dec)
	}
	return out
}

func TestTOCThresholdIsInclusive(t *testing.T) {
	if tocBlock(renderDocWithTOC(t, numberedSections(7), 8)) != "" {
		t.Errorf("7 headings is below a threshold of 8: no table of contents")
	}
	if tocBlock(renderDocWithTOC(t, numberedSections(8), 8)) == "" {
		t.Errorf("8 headings meets a threshold of 8: a table of contents is expected")
	}
	if n := strings.Count(renderDocWithTOC(t, numberedSections(12), 8), `<nav class="toc"`); n != 1 {
		t.Errorf("exactly one table of contents is expected, got %d", n)
	}
}

func TestTOCOffByDefaultAndWhenZero(t *testing.T) {
	src := numberedSections(20)
	if tocBlock(renderDocWithTOC(t, src, 0)) != "" {
		t.Errorf("a threshold of 0 means no table of contents however long the page")
	}
	// the plain entry point never adds one
	repo, _ := core.Init(t.TempDir())
	plain, err := renderMarkdown(repo, "", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	if tocBlock(string(plain)) != "" {
		t.Errorf("renderMarkdown must not add a table of contents")
	}
}

func TestTOCSitsAfterTheIntroAndBeforeTheFirstSection(t *testing.T) {
	html := renderDocWithTOC(t, numberedSections(4), 3)
	intro := strings.Index(html, "An introduction.")
	toc := strings.Index(html, `<nav class="toc"`)
	first := strings.Index(html, `<h2 id="section-1"`)
	if !(intro >= 0 && toc > intro && first > toc) {
		t.Errorf("want intro (%d) < toc (%d) < first section (%d):\n%s", intro, toc, first, html)
	}
	// the title is the page's own heading, not an entry in its contents
	if strings.Contains(tocBlock(html), ">Title<") {
		t.Errorf("the h1 title must not be listed:\n%s", tocBlock(html))
	}
}

func TestTOCListsH2AndH3AndNestsTheH3(t *testing.T) {
	src := "# T\n\n## One\n\n### One A\n\n### One B\n\n#### Too deep\n\n## Two\n"
	html := renderDocWithTOC(t, src, 3)
	toc := tocBlock(html)

	if got := strings.Join(tocFragments(t, html), ","); got != "one,one-a,one-b,two" {
		t.Errorf("entries = %q, want one,one-a,one-b,two\n%s", got, toc)
	}
	if strings.Contains(toc, "Too deep") {
		t.Errorf("h4 and below are not listed:\n%s", toc)
	}
	// "One" owns a nested list holding its two h3s, and "Two" is a sibling of "One"
	want := regexp.MustCompile(`(?s)<li><a href="#one">One</a>\s*<ul>\s*<li><a href="#one-a">One A</a></li>\s*<li><a href="#one-b">One B</a></li>\s*</ul>\s*</li>\s*<li><a href="#two">Two</a></li>`)
	if !want.MatchString(toc) {
		t.Errorf("h3s must nest under the h2 before them:\n%s", toc)
	}
}

func TestTOCH3WithNoH2BeforeItIsStillListed(t *testing.T) {
	html := renderDocWithTOC(t, "### Orphan\n\n## Later\n\n## Last\n", 3)
	if got := strings.Join(tocFragments(t, html), ","); got != "orphan,later,last" {
		t.Errorf("an h3 before any h2 must be listed at the top level, got %q\n%s", got, tocBlock(html))
	}
}

func TestTOCLinksMatchHeadingIdsInOrderIncludingDuplicates(t *testing.T) {
	src := "# T\n\n## Intro\n\n## Intro\n\n## Intro\n\n## snake_case names\n\n## Café au lait\n\n## 日本語 の 見出し\n"
	html := renderDocWithTOC(t, src, 3)

	var wantIDs []string
	for _, id := range headingIDs(html) {
		if id != "t" { // the h1
			wantIDs = append(wantIDs, id)
		}
	}
	got := tocFragments(t, html)
	if strings.Join(got, ",") != strings.Join(wantIDs, ",") {
		t.Errorf("TOC fragments %v must equal the headings' ids %v", got, wantIDs)
	}
	if strings.Join(got[:3], ",") != "intro,intro-1,intro-2" {
		t.Errorf("duplicate headings keep their numbered ids, got %v", got[:3])
	}
	// non-ASCII ids are percent-encoded in the href, which browsers resolve
	if !strings.Contains(tocBlock(html), `href="#caf%C3%A9-au-lait"`) {
		t.Errorf("a non-ASCII id must be percent-encoded in the link:\n%s", tocBlock(html))
	}
}

func TestTOCLabelsAreVisibleTextEscapedAndNeverPermalink(t *testing.T) {
	src := "# T\n\n## The `git track web` command\n\n## Emphasis *here* and [a link](http://x.example)\n\n## a <b>bold</b> & \"quoted\" claim\n"
	toc := tocBlock(renderDocWithTOC(t, src, 3))

	if strings.Contains(toc, "Permalink") {
		t.Errorf("the permalink word must never reach the table of contents:\n%s", toc)
	}
	for _, want := range []string{">The git track web command<", ">Emphasis here and a link<"} {
		if !strings.Contains(toc, want) {
			t.Errorf("label %q missing; labels are the heading's visible text:\n%s", want, toc)
		}
	}
	if strings.Contains(toc, "<code>") || strings.Contains(toc, "<em>") || strings.Contains(toc, "x.example") {
		t.Errorf("a TOC label is plain text, with no markup or link destination:\n%s", toc)
	}
	if strings.Contains(toc, "<b>") || strings.Contains(toc, `"quoted"`) {
		t.Errorf("heading text must be escaped inside the table of contents:\n%s", toc)
	}
}

func TestTOCImageOnlyHeadingFallsBackToItsId(t *testing.T) {
	html := renderDocWithTOC(t, "## ![](x.png)\n\n## B\n\n## C\n", 3)
	if !strings.Contains(tocBlock(html), `<a href="#section">section</a>`) {
		t.Errorf("a heading with no visible text is labelled by its id:\n%s", tocBlock(html))
	}
}

func TestTOCOnlyCountsTopLevelHeadingsOfTheRightLevel(t *testing.T) {
	// three real sections, plus headings that must not count: in a quote, in a
	// list, inside a code fence, an h1, and an h4
	src := "# T\n\n## A\n\n## B\n\n## C\n\n> ## In a quote\n\n- ### In a list\n\n```md\n## Fake heading\n```\n\n#### Deep\n"
	if tocBlock(renderDocWithTOC(t, src, 4)) != "" {
		t.Errorf("only 3 top-level h2/h3 headings exist; threshold 4 must not be met")
	}
	html := renderDocWithTOC(t, src, 3)
	if got := strings.Join(tocFragments(t, html), ","); got != "a,b,c" {
		t.Errorf("entries = %q, want a,b,c\n%s", got, tocBlock(html))
	}
	// the nested headings still get ids and permalinks
	if ids := strings.Join(headingIDs(html), ","); !strings.Contains(ids, "in-a-quote") || !strings.Contains(ids, "in-a-list") {
		t.Errorf("nested headings keep their ids, got %q", ids)
	}
}

func TestTOCCoexistsWithPermalinksAndHighlightedFences(t *testing.T) {
	repo, _ := core.Init(t.TempDir())
	src := "# T\n\nIntro.\n\n## A\n\n```go\nfunc main() {}\n```\n\n## B\n\n## C\n"
	res, err := renderMarkdownWith(markdownOpts{repo: repo, hl: mustHighlighter(t, "github"), tocMin: 3}, src)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	html := string(res.HTML)
	if tocBlock(html) == "" || !strings.Contains(html, `<pre class="chroma">`) {
		t.Errorf("both the table of contents and the highlighted fence are expected:\n%s", html)
	}
	if !strings.Contains(html, `class="anchor"`) {
		t.Errorf("headings keep their permalinks:\n%s", html)
	}
	if d := maxAnchorDepth(html); d != 1 {
		t.Errorf("links must not nest (depth %d)", d)
	}
}

func TestTOCIsAccessibleNavigation(t *testing.T) {
	toc := tocBlock(renderDocWithTOC(t, numberedSections(3), 3))
	if !strings.HasPrefix(toc, `<nav class="toc" aria-label="On this page">`) {
		t.Errorf("the contents should be a labelled navigation landmark:\n%s", toc)
	}
	if !strings.Contains(toc, `<p class="toc-title">On this page</p>`) {
		t.Errorf("the contents need a visible title:\n%s", toc)
	}
}

// ---- docs section ----------------------------------------------------------

func docsTOCEntries(page string) int { return strings.Count(tocBlock(page), `<li><a href="#`) }

func TestDocsPagesGetATOCOnlyWhenLong(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionDocs)
	writeRepoFile(t, repo, "docs/short.md", numberedSections(defaultTOCMin-1))
	writeRepoFile(t, repo, "docs/long.md", numberedSections(defaultTOCMin))
	out := generate(t, repo)

	short, long := readOut(t, out, "docs/short.html"), readOut(t, out, "docs/long.html")
	if tocBlock(short) != "" {
		t.Errorf("a page just under the default threshold must have no table of contents")
	}
	if got := docsTOCEntries(long); got != defaultTOCMin {
		t.Errorf("a page at the threshold lists all %d sections, got %d:\n%s", defaultTOCMin, got, tocBlock(long))
	}
	// the sidebar and the page contents are different things and both are present
	if !strings.Contains(long, `class="docs-nav"`) || !strings.Contains(long, `<nav class="toc"`) {
		t.Errorf("a long docs page carries the sidebar and its own table of contents")
	}
}

func TestDocsTOCConfig(t *testing.T) {
	cases := []struct {
		setting string
		page    int // sections on the page
		want    bool
	}{
		{"off", 30, false},
		{"OFF", 30, false},
		{"0", 30, false},
		{"3", 3, true},
		{"3", 2, false},
		{"100", 30, false},
		{"-2", defaultTOCMin, true},      // negative: ignored, default applies
		{"-2", defaultTOCMin - 1, false}, //
		{"lots", defaultTOCMin, true},    // garbage: default applies
		{"lots", defaultTOCMin - 1, false},
	}
	for _, c := range cases {
		repo, err := core.Init(t.TempDir())
		if err != nil {
			t.Fatalf("Init: %v", err)
		}
		enableSections(t, repo, sectionDocs)
		writeSiteFile(t, repo, "docs/toc", c.setting+"\n")
		writeRepoFile(t, repo, "docs/page.md", numberedSections(c.page))
		page := readOut(t, generate(t, repo), "docs/page.html")
		if got := tocBlock(page) != ""; got != c.want {
			t.Errorf("toc=%q with %d sections: table of contents = %v, want %v", c.setting, c.page, got, c.want)
		}
	}
}

func TestOnlyDocsPagesGetATOC(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	long := numberedSections(15)
	enableSections(t, repo, sectionPortal, sectionBlog, sectionReleases)
	writeSiteFile(t, repo, "portal/front.md", long)
	writeSiteFile(t, repo, "blog/post.md", "---\ntitle: Post\ndate: 2025-01-01\n---\n"+long)
	mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", long)
	out := generate(t, repo)
	for _, rel := range []string{"index.html", "blog/post.html", "releases/v1-0-0.html"} {
		if tocBlock(readOut(t, out, rel)) != "" {
			t.Errorf("%s: only docs pages get a table of contents", rel)
		}
	}
}

func TestDocsTOCLinksPointAtRealIdsOnTheSamePage(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionDocs)
	writeRepoFile(t, repo, "docs/page.md", "# P\n\nintro\n\n## Dup\n\n## Dup\n\n"+strings.Repeat("## More\n\n", 3)+"## snake_case\n\n## Ünïcode\n\n### Child\n")
	page := readOut(t, generate(t, repo), "docs/page.html")

	have := map[string]bool{}
	for _, id := range headingIDs(page) {
		have[id] = true
	}
	frags := tocFragments(t, page)
	if len(frags) == 0 {
		t.Fatalf("expected a table of contents on this page")
	}
	for _, f := range frags {
		if !have[f] {
			t.Errorf("TOC links to #%s but the page has no such id (ids: %v)", f, headingIDs(page))
		}
	}
}
