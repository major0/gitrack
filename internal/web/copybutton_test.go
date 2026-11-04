package web

import (
	"regexp"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// hasCopyScript reports whether a page loads the copy-button script.
func hasCopyScript(page string) bool { return strings.Contains(page, copyAsset+`"></script>`) }

func TestCopyScriptContainsNothingThatCouldBreakOrLeakOut(t *testing.T) {
	s := copyScript
	if strings.Contains(strings.ToLower(s), "</script") {
		t.Errorf("the script must not contain a closing script tag")
	}
	for _, bad := range []string{"http:", "https:", "//cdn", "src=", "import(", "fetch(", "XMLHttpRequest", "eval(", "innerHTML"} {
		if strings.Contains(s, bad) {
			t.Errorf("the script must make no network request and build no markup from strings: found %q", bad)
		}
	}
	if !strings.HasPrefix(s, "(function () {") || !strings.HasSuffix(s, "})();") {
		t.Errorf("the script should be one self-contained function so it leaks no globals")
	}
}

func TestHasCodeBlock(t *testing.T) {
	cases := map[string]struct {
		src  string
		want bool
	}{
		"fenced":              {"```go\nx := 1\n```\n", true},
		"fenced, no language": {"```\nplain\n```\n", true},
		"indented":            {"text\n\n    indented code\n", true},
		"inline code only":    {"use `git track` here\n", false},
		"no code":             {"# Title\n\njust words\n", false},
		// authored text can never forge the markers: text is escaped and raw HTML is off
		"prose naming the tag":  {"write `<pre><code>` to start a block\n", false},
		"raw HTML pre":          {"<pre><code>sneaky</code></pre>\n", false},
		"raw HTML chroma class": {`<pre class="chroma">sneaky</pre>` + "\n", false},
	}
	for name, c := range cases {
		res := renderWith(t, nil, c.src)
		if res.HasCode != c.want {
			t.Errorf("%s: HasCode = %v, want %v\n%s", name, res.HasCode, c.want, res.HTML)
		}
	}
	// a highlighted fence counts too (it is rendered as <pre class="chroma">)
	if res := renderWith(t, mustHighlighter(t, "github"), "```go\nx := 1\n```\n"); !res.HasCode {
		t.Errorf("a highlighted fence is a code block")
	}
}

func TestScriptsForOnlyListsTheCopyScriptWhenThereIsCode(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	th, err := loadTheme(repo, "")
	if err != nil {
		t.Fatalf("loadTheme: %v", err)
	}
	sc := &siteContext{repo: repo, theme: th}
	if got := sc.scriptsFor(extraFor(true, false)); len(got) != 0 {
		t.Errorf("no code: no script, got %v", got)
	}
	if got := sc.scriptsFor(extraFor(false, true)); len(got) != 1 || got[0] != copyAsset {
		t.Errorf("code present and the button on: the copy script is expected, got %v", got)
	}
	writeSiteFile(t, repo, "copy", "off\n")
	off := &siteContext{repo: repo, theme: th}
	if got := off.scriptsFor(extraFor(true, true)); len(got) != 0 {
		t.Errorf("button off: no script, got %v", got)
	}
}

func TestCopySettingValues(t *testing.T) {
	for value, wantOn := range map[string]bool{
		"off": false, "OFF": false, "Off": false,
		"on": true, "yes": true, "": true, "garbage": true, "0": true,
	} {
		repo, err := core.Init(t.TempDir())
		if err != nil {
			t.Fatalf("Init: %v", err)
		}
		if value != "" {
			writeSiteFile(t, repo, "copy", value+"\n")
		}
		if got := (&siteContext{repo: repo}).copyEnabled(); got != wantOn {
			t.Errorf("copy=%q: enabled = %v, want %v", value, got, wantOn)
		}
	}
}

func TestPagesGetTheScriptOnlyWhereThereIsCode(t *testing.T) {
	repo, dir := newSourceRepo(t)
	fence := "\n```sh\nmake\n```\n"
	enableSections(t, repo, sectionIssues, sectionPortal, sectionBlog, sectionDocs, sectionReleases, sectionSource)
	writeSiteFile(t, repo, "portal/front.md", "# Home\n"+fence)
	writeSiteFile(t, repo, "portal/about.md", "# About\n\nno code\n")
	writeSiteFile(t, repo, "blog/post.md", "---\ntitle: Post\ndate: 2025-01-01\n---\n"+fence)
	writeSiteFile(t, repo, "blog/plain.md", "---\ntitle: Plain\ndate: 2025-01-02\n---\n\nno code\n")
	writeRepoFile(t, repo, "docs/guide.md", "# Guide\n"+fence)
	writeRepoFile(t, repo, "docs/plain.md", "# Plain\n\nno code\n")
	id, _ := repo.Create(core.NewTrackOpts{Title: "A track", Description: "preformatted text"})
	mkRelease(t, repo, "v1.0.0", "2025-01-01T00:00:00Z", "## Notes\n"+fence)
	mkRelease(t, repo, "v0.9.0", "2024-01-01T00:00:00Z", "just words")
	writeRepoText(t, dir, "main.go", "package main\n")
	writeRepoText(t, dir, "empty.txt", "")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")
	out := generate(t, repo)

	// issues is listed first, so it owns index.html and the portal front page is
	// rendered as portal/front.html
	with := []string{"portal/front.html", "blog/post.html", "docs/guide.html", "releases/v1-0-0.html", "source/blob/main.go/index.html"}
	without := []string{
		"portal/about.html", "blog/plain.html", "blog/index.html", "docs/plain.html", "docs/index.html",
		"releases/v0-9-0.html", "releases/index.html",
		"tracks.html", "issues.html", "board.html", "track/" + id + ".html",
		"source/index.html", "source/log.html", "source/blob/empty.txt/index.html",
	}
	for _, rel := range with {
		page := readOut(t, out, rel)
		if !hasCopyScript(page) {
			t.Errorf("%s holds a code block and should carry the copy script", rel)
		}
		if n := strings.Count(page, "<script"); n != 1 {
			t.Errorf("%s: want exactly one script element, got %d", rel, n)
		}
		// a script file, never inline, and the last thing before the document ends
		if !regexp.MustCompile(`(?s)</footer>\n<script src="[./]*assets/gitrack\.js"></script>\n</body></html>\s*$`).MatchString(page) {
			t.Errorf("%s: the script should be a file loaded after the footer, at the end of the body", rel)
		}
	}
	for _, rel := range without {
		page, err := readOutMaybe(out, rel)
		if err != nil {
			continue // a page that does not exist in this layout
		}
		if hasCopyScript(page) {
			t.Errorf("%s has no code block and must not carry the copy script", rel)
		}
	}
	// a page without the script ends exactly as pages did before the feature existed
	about := readOut(t, out, "portal/about.html")
	if !strings.HasSuffix(about, "</footer>\n</body></html>") {
		t.Errorf("a page without the script must be unchanged at its end:\n...%s", about[max(0, len(about)-120):])
	}
}

func TestCopyOffRemovesTheScriptEverywhere(t *testing.T) {
	repo, dir := newSourceRepo(t)
	enableSections(t, repo, sectionDocs, sectionSource)
	writeRepoFile(t, repo, "docs/guide.md", "# Guide\n\n```sh\nmake\n```\n")
	writeRepoText(t, dir, "main.go", "package main\n")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")

	on := generate(t, repo)
	if !hasCopyScript(readOut(t, on, "docs/guide.html")) || !hasCopyScript(readOut(t, on, "source/blob/main.go/index.html")) {
		t.Fatalf("precondition: the script is present by default")
	}
	writeSiteFile(t, repo, "copy", "off\n")
	off := generate(t, repo)
	for _, rel := range []string{"docs/guide.html", "source/blob/main.go/index.html"} {
		page := readOut(t, off, rel)
		if strings.Contains(page, "<script") {
			t.Errorf("%s: copy=off must leave no script on the page", rel)
		}
	}
}

func TestCopyScriptCoexistsWithHighlightingAndTOC(t *testing.T) {
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionDocs)
	writeRepoFile(t, repo, "docs/guide.md", "# Guide\n\n"+numberedSectionsWithFence(9))
	out := generate(t, repo)
	page := readOut(t, out, "docs/guide.html")
	for what, ok := range map[string]bool{
		"the highlighted block":   strings.Contains(page, `<pre class="chroma">`),
		"the token stylesheet":    linksHighlightCSS(page),
		"the table of contents":   strings.Contains(page, `<nav class="toc"`),
		"the copy script":         hasCopyScript(page),
		"the button's own styles": strings.Contains(readOut(t, out, themeCSSAsset), ".copy-btn"),
	} {
		if !ok {
			t.Errorf("expected %s on a long docs page with code", what)
		}
	}
}

// numberedSectionsWithFence is n h2 sections, the first holding a code fence.
func numberedSectionsWithFence(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("## Section " + string(rune('0'+i)) + "\n\ntext\n\n")
		if i == 1 {
			b.WriteString("```go\nfunc main() {}\n```\n\n")
		}
	}
	return b.String()
}
