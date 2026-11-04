package web

import (
	"regexp"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func renderWith(t *testing.T, hl *highlighter, src string) mdResult {
	t.Helper()
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	res, err := renderMarkdownWith(markdownOpts{repo: repo, hl: hl}, src)
	if err != nil {
		t.Fatalf("renderMarkdownWith: %v", err)
	}
	return res
}

func TestFencedCodeIsHighlighted(t *testing.T) {
	res := renderWith(t, mustHighlighter(t, "github"), "```go\nfunc main() {}\n```\n")
	html := string(res.HTML)
	if !strings.Contains(html, `<pre class="chroma"><code class="language-go">`) {
		t.Errorf("a go fence should become a chroma block:\n%s", html)
	}
	if !strings.Contains(html, `<span class="kd">func</span>`) {
		t.Errorf("tokens should be classed:\n%s", html)
	}
	if !res.Highlighted {
		t.Errorf("a page with a highlighted block must report it, so it links the stylesheet")
	}
}

func TestFencesLeftPlainKeepDefaultRendering(t *testing.T) {
	h := mustHighlighter(t, "github")
	cases := map[string]struct {
		hl   *highlighter
		src  string
		want string
	}{
		"unknown language":      {h, "```zzzunknown\nx\n```\n", `<pre><code class="language-zzzunknown">`},
		"no language":           {h, "```\nx\n```\n", `<pre><code>`},
		"plain text language":   {h, "```text\nx\n```\n", `<pre><code class="language-text">`},
		"highlighting disabled": {nil, "```go\nx := 1\n```\n", `<pre><code class="language-go">`},
	}
	for name, c := range cases {
		res := renderWith(t, c.hl, c.src)
		html := string(res.HTML)
		if !strings.Contains(html, c.want) {
			t.Errorf("%s: want %q in:\n%s", name, c.want, html)
		}
		if strings.Contains(html, "chroma") || res.Highlighted {
			t.Errorf("%s: must not be highlighted or need the stylesheet:\n%s", name, html)
		}
	}
}

func TestFenceLanguageAliasesAndInfoStringWords(t *testing.T) {
	h := mustHighlighter(t, "github")
	for _, lang := range []string{"sh", "bash", "yaml", "json", "markdown", "diff", "python", "html"} {
		res := renderWith(t, h, "```"+lang+"\nx\n```\n")
		if !strings.Contains(string(res.HTML), `<pre class="chroma"><code class="language-`+lang+`">`) {
			t.Errorf("%s should be recognized:\n%s", lang, res.HTML)
		}
	}
	// extra words after the language are ignored
	res := renderWith(t, h, "```go title=main.go\nvar x = 1\n```\n")
	if !strings.Contains(string(res.HTML), `<code class="language-go">`) {
		t.Errorf("the first word is the language:\n%s", res.HTML)
	}
}

func TestHighlightedFenceStaysInertForHTML(t *testing.T) {
	h := mustHighlighter(t, "github")
	for name, src := range map[string]string{
		"html":      "```html\n<script>alert(1)</script>\n```\n",
		"go string": "```go\ns := \"<script>alert(1)</script>\"\n```\n",
	} {
		html := string(renderWith(t, h, src).HTML)
		if strings.Contains(html, "<script>") {
			t.Errorf("%s: code must be escaped, never emitted as markup:\n%s", name, html)
		}
		if !strings.Contains(html, "&lt;") {
			t.Errorf("%s: escaped angle bracket expected:\n%s", name, html)
		}
	}
}

func TestFencesInListsAndQuotesAreHighlighted(t *testing.T) {
	h := mustHighlighter(t, "github")
	html := string(renderWith(t, h, "- item\n\n  ```go\n  func f() {}\n  ```\n\n> ```go\n> var x = 1\n> ```\n").HTML)
	if n := strings.Count(html, `<pre class="chroma">`); n != 2 {
		t.Errorf("both nested fences should be highlighted, got %d:\n%s", n, html)
	}
}

func TestRefInsideHighlightedFenceIsNotLinked(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, _ := repo.Create(core.NewTrackOpts{Title: "Login flow"})
	slug, _ := repo.GetField(id, "slug")
	res, err := renderMarkdownWith(markdownOpts{repo: repo, hl: mustHighlighter(t, "github")},
		"```go\n// see #"+slug+"\n```\n")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	html := string(res.HTML)
	if strings.Contains(html, "<a ") {
		t.Errorf("a #ref in code must stay literal:\n%s", html)
	}
	if !strings.Contains(html, "#"+slug) {
		t.Errorf("the ref text must still be there:\n%s", html)
	}
}

func TestFenceBudgetFallsBackToPlainAndSkipsSpendNothing(t *testing.T) {
	h := mustHighlighter(t, "github")
	h.remaining = 30
	first := "func a() {}\n"                    // 12 bytes
	tooBig := "func b() { println(1, 2, 3) }\n" // 29 bytes, more than the 18 left
	src := "```go\n" + first + "```\n\n```go\n" + tooBig + "```\n\n```go\n" + first + "```\n"
	html := string(renderWith(t, h, src).HTML)

	if n := strings.Count(html, `<pre class="chroma">`); n != 2 {
		t.Errorf("the first and third fit and should be highlighted, got %d:\n%s", n, html)
	}
	if !strings.Contains(html, `<pre><code class="language-go">func b()`) {
		t.Errorf("the block that no longer fits should be plain:\n%s", html)
	}
	// a block over the per-text cap is plain regardless of budget
	h2 := mustHighlighter(t, "github")
	huge := strings.Repeat("x := 1\n", maxHighlightBytes/6)
	res := renderWith(t, h2, "```go\n"+huge+"```\n")
	if strings.Contains(string(res.HTML), "chroma") || res.Highlighted {
		t.Errorf("an oversized block must stay plain with no stylesheet")
	}
}

func TestMarkdownWithoutHighlighterMatchesPlainRenderer(t *testing.T) {
	repo, _ := core.Init(t.TempDir())
	src := "# Title\n\ntext\n\n```go\nx := 1\n```\n\n- a\n"
	plain, err := renderMarkdown(repo, "", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	with, err := renderMarkdownWith(markdownOpts{repo: repo}, src)
	if err != nil {
		t.Fatalf("renderMarkdownWith: %v", err)
	}
	if plain != with.HTML {
		t.Errorf("no highlighter must mean exactly the plain renderer's output")
	}
}

func TestHeadingsSurviveAlongsideHighlightedFences(t *testing.T) {
	html := string(renderWith(t, mustHighlighter(t, "github"), "# Install\n\n```sh\nmake\n```\n\n## Usage\n").HTML)
	if got := strings.Join(headingIDs(html), ","); got != "install,usage" {
		t.Errorf("ids = %q\n%s", got, html)
	}
}

func TestSiteHighlighterIsSharedAndBuiltOnce(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	sc := &siteContext{repo: repo}
	a, err := sc.highlighter()
	if err != nil || a == nil {
		t.Fatalf("highlighter() = %v, %v", a, err)
	}
	b, _ := sc.highlighter()
	if a != b {
		t.Errorf("the highlighter must be one shared instance so the budget is shared")
	}
	writeSiteFile(t, repo, "highlight", "off\n")
	off := &siteContext{repo: repo}
	if h, err := off.highlighter(); h != nil || err != nil {
		t.Errorf(`"off" should give no highlighter, got %v, %v`, h, err)
	}
}

// linksHighlightCSS reports whether a page links the token stylesheet.
func linksHighlightCSS(page string) bool {
	return regexp.MustCompile(`<link rel="stylesheet" href="[^"]*assets/highlight\.css">`).MatchString(page)
}

// tokenCSS returns the site's token stylesheet, failing the test if it was not
// written. There is one per site: it is not repeated in every page.
func tokenCSS(t *testing.T, out string) string {
	t.Helper()
	return readOut(t, out, highlightAsset)
}

func TestMarkdownSectionsAttachHighlightingOnlyWhereUsed(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	fence := "\n```go\nfunc main() {}\n```\n"
	enableSections(t, repo, sectionPortal, sectionBlog, sectionDocs, sectionReleases)
	writeSiteFile(t, repo, "portal/front.md", "# Home\n"+fence)
	writeSiteFile(t, repo, "portal/about.md", "# About\n\nno code here\n")
	writeSiteFile(t, repo, "blog/post.md", "---\ntitle: Post\ndate: 2025-01-01\n---\n"+fence)
	writeSiteFile(t, repo, "blog/plain.md", "---\ntitle: Plain\ndate: 2025-01-02\n---\n\nno code\n")
	writeRepoFile(t, repo, "docs/guide.md", "# Guide\n"+fence)
	writeRepoFile(t, repo, "docs/plain.md", "# Plain\n\nno code\n")
	mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", "## Notes\n"+fence)
	mkRelease(t, repo, "v0.9.0", "2024-01-01T00:00:00Z", "just words")

	out := generate(t, repo)
	withCode := []string{"index.html", "blog/post.html", "docs/guide.html", "releases/v1-0-0.html"}
	without := []string{"portal/about.html", "blog/plain.html", "docs/plain.html", "releases/v0-9-0.html"}
	for _, rel := range withCode {
		page := readOut(t, out, rel)
		if !strings.Contains(page, `<pre class="chroma">`) {
			t.Errorf("%s: the fence should be highlighted", rel)
		}
		if !linksHighlightCSS(page) {
			t.Errorf("%s: a highlighted page must link the token stylesheet", rel)
		}
	}
	if css := tokenCSS(t, out); !strings.Contains(css, ".chroma") {
		t.Errorf("assets/highlight.css should be the chroma stylesheet")
	}
	for _, rel := range without {
		page := readOut(t, out, rel)
		if linksHighlightCSS(page) || strings.Contains(page, "chroma") {
			t.Errorf("%s: a page without code must not link the token stylesheet", rel)
		}
	}
}

func TestOneHighlightSettingGovernsSourceAndMarkdown(t *testing.T) {
	repo, dir := newSourceRepo(t)
	writeRepoText(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writeRepoText(t, dir, "docs/guide.md", "# Guide\n\n```go\nfunc main() {}\n```\n")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")
	enableSections(t, repo, sectionDocs, sectionSource)

	blob := func(out string) string { return readOut(t, out, "source/blob/main.go/index.html") }
	doc := func(out string) string { return readOut(t, out, "docs/guide.html") }

	// default: both highlighted, and from the very same stylesheet
	out := generate(t, repo)
	if !strings.Contains(blob(out), `class="blob chroma"`) || !strings.Contains(doc(out), `<pre class="chroma">`) {
		t.Fatalf("by default both the source page and the docs fence should be highlighted")
	}
	if !linksHighlightCSS(blob(out)) || !linksHighlightCSS(doc(out)) {
		t.Errorf("source and docs must link the same single token stylesheet")
	}
	github := tokenCSS(t, out)

	// a named style applies to both, through that one stylesheet
	writeSiteFile(t, repo, "highlight", "monokai\n")
	out = generate(t, repo)
	if !linksHighlightCSS(blob(out)) || !linksHighlightCSS(doc(out)) {
		t.Errorf("source and docs must still share the stylesheet after changing the style")
	}
	if mono := tokenCSS(t, out); mono == github {
		t.Errorf("changing the style must change the stylesheet")
	}

	// off turns both off
	writeSiteFile(t, repo, "highlight", "off\n")
	out = generate(t, repo)
	if strings.Contains(blob(out), "chroma") || strings.Contains(doc(out), "chroma") {
		t.Errorf("highlight=off must turn off the source page and the docs fence")
	}
	if linksHighlightCSS(doc(out)) || linksHighlightCSS(blob(out)) {
		t.Errorf("no token stylesheet link when off")
	}
}

func TestLegacySourceHighlightLocationIsNoLongerRead(t *testing.T) {
	repo, dir := newSourceRepo(t)
	writeRepoText(t, dir, "main.go", "package main\n")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")
	enableSections(t, repo, sectionSource)
	// the setting used to live under source/; it must not silently still apply
	writeSiteFile(t, repo, "source/highlight", "off\n")
	out := generate(t, repo)
	if !strings.Contains(readOut(t, out, "source/blob/main.go/index.html"), `class="blob chroma"`) {
		t.Errorf("source/highlight was replaced by the site-wide highlight file and must be ignored")
	}
}
