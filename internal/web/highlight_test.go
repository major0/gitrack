package web

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// codeLines splits rendered code into its per-line spans.
func codeLines(t *testing.T, html string) []string {
	t.Helper()
	return strings.Split(strings.TrimRight(html, "\n"), "\n")
}

func mustHighlighter(t *testing.T, style string) *highlighter {
	t.Helper()
	h, err := newHighlighter(style)
	if err != nil || h == nil {
		t.Fatalf("newHighlighter(%q) = %v, %v", style, h, err)
	}
	return h
}

func TestHighlightGoColorsTokensAndKeepsLines(t *testing.T) {
	h := mustHighlighter(t, "github")
	src := "package main\n\n/* a block\n   comment that\n   spans lines */\nfunc main() {\n\tprintln(\"<script>alert(1)</script>\")\n}\n"
	res := renderCode([]byte(src), h, "main.go")
	if !res.Highlighted || res.Note != "" {
		t.Fatalf("a .go file should be highlighted with no note, got %+v", res)
	}
	html := string(res.HTML)

	if !strings.Contains(html, `<span class="kd">func</span>`) {
		t.Errorf("the func keyword should be a classed token:\n%s", html)
	}

	lines := codeLines(t, html)
	want := strings.Count(src, "\n") // the source ends in a newline, so this is the line count
	if len(lines) != want {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), want, html)
	}
	for i, line := range lines {
		wantID := `<span class="src-line" id="L` + strconv.Itoa(i+1) + `">`
		if !strings.HasPrefix(line, wantID) {
			t.Errorf("line %d should start %q, got %q", i+1, wantID, line)
		}
		// every line, including those inside the multi-line comment, is self-contained
		if open, closed := strings.Count(line, "<span"), strings.Count(line, "</span>"); open != closed {
			t.Errorf("line %d has unbalanced spans (%d open, %d closed): %q", i+1, open, closed, line)
		}
	}

	// code that contains HTML is inert: escaped, never emitted as markup
	if strings.Contains(html, "<script>") {
		t.Errorf("a script tag in source must be escaped:\n%s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("escaped script tag expected:\n%s", html)
	}
}

func TestHighlightLineCountIsStableAcrossEndings(t *testing.T) {
	h := mustHighlighter(t, "github")
	for name, src := range map[string]string{
		"trailing newline":    "package x\nfunc f() {}\n",
		"no trailing newline": "package x\nfunc f() {}",
		"crlf":                "package x\r\nfunc f() {}\r\n",
		"single line":         "package x",
	} {
		plain := renderCode([]byte(src), nil, "x.go")
		hl := renderCode([]byte(src), h, "x.go")
		pn := strings.Count(string(plain.HTML), `class="src-line"`)
		hn := strings.Count(string(hl.HTML), `class="src-line"`)
		if pn != hn {
			t.Errorf("%s: plain has %d lines but highlighted has %d", name, pn, hn)
		}
	}
}

func TestHighlightFallsBackToPlainWithAllLines(t *testing.T) {
	h := mustHighlighter(t, "github")
	src := "line one\nline two\nline three\n"

	cases := map[string]struct {
		h    *highlighter
		path string
		note string
	}{
		"unknown extension":     {h, "data.zzzunknown", ""},
		"plain text lexer":      {h, "notes.txt", ""},
		"highlighting disabled": {nil, "main.go", ""},
	}
	for name, c := range cases {
		res := renderCode([]byte(src), c.h, c.path)
		if res.Highlighted {
			t.Errorf("%s: should not be highlighted", name)
		}
		if res.Note != c.note {
			t.Errorf("%s: note = %q, want %q", name, res.Note, c.note)
		}
		if n := strings.Count(string(res.HTML), `class="src-line"`); n != 3 {
			t.Errorf("%s: plain fallback must keep all 3 lines, got %d", name, n)
		}
	}

	// well-known file NAMES with no extension are recognized (the guide names both)
	for name, src := range map[string]string{
		"Makefile":   "all:\n\techo hi\n",
		"Dockerfile": "FROM scratch\nCOPY a /a\n",
	} {
		if res := renderCode([]byte(src), h, name); !res.Highlighted {
			t.Errorf("%s should be recognized by name", name)
		}
	}
}

func TestHighlightSkipsLargeFiles(t *testing.T) {
	h := mustHighlighter(t, "github")
	big := strings.Repeat("var x = 1\n", maxHighlightBytes/5) // comfortably over the cap
	res := renderCode([]byte(big), h, "big.go")
	if res.Highlighted {
		t.Errorf("a file over the cap must not be highlighted")
	}
	if !strings.Contains(res.Note, "not highlighted: larger than") {
		t.Errorf("note = %q, want a too-large note", res.Note)
	}
	if n := strings.Count(string(res.HTML), `class="src-line"`); n != maxHighlightBytes/5 {
		t.Errorf("every line must still be shown, got %d", n)
	}
}

func TestHighlightGivesUpWhenOutOfTime(t *testing.T) {
	h := mustHighlighter(t, "github")
	old := highlightBudget
	highlightBudget = -time.Second // already expired
	defer func() { highlightBudget = old }()

	// enough tokens that the deadline is checked
	src := strings.Repeat("a := 1\n", 500)
	res := renderCode([]byte(src), h, "x.go")
	if res.Highlighted {
		t.Errorf("an exhausted budget must fall back to plain text")
	}
	if !strings.Contains(res.Note, "could not be tokenized in time") {
		t.Errorf("note = %q, want a timeout note", res.Note)
	}
	if n := strings.Count(string(res.HTML), `class="src-line"`); n != 500 {
		t.Errorf("every line must still be shown, got %d", n)
	}
}

func TestHighlightTotalBudgetIsDeterministic(t *testing.T) {
	h := mustHighlighter(t, "github")
	small := []byte("package a\n")           // 10 bytes
	big := []byte("package b\n// padding\n") // 21 bytes
	h.remaining = 25

	first := renderCode(small, h, "a.go") // 10 spent, 15 left
	if !first.Highlighted {
		t.Fatalf("the first file fits the budget and should be highlighted")
	}
	second := renderCode(big, h, "b.go") // 21 > 15 left
	if second.Highlighted || !strings.Contains(second.Note, "highlighting limit was reached") {
		t.Errorf("a file that no longer fits must be plain with a note, got %+v", second)
	}
	if n := strings.Count(string(second.HTML), `class="src-line"`); n != 2 {
		t.Errorf("the plain fallback must keep every line, got %d", n)
	}
	// a skipped file spends nothing, so a later file that does fit is still highlighted
	third := renderCode(small, h, "c.go")
	if !third.Highlighted {
		t.Errorf("a smaller file that still fits should be highlighted after a skip")
	}
	// a file in no known language never touches the budget
	before := h.remaining
	renderCode([]byte("plain\n"), h, "x.zzzunknown")
	if h.remaining != before {
		t.Errorf("an unhighlighted file must not spend budget: %d -> %d", before, h.remaining)
	}
}

func TestNewHighlighterStyles(t *testing.T) {
	if h, err := newHighlighter("off"); h != nil || err != nil {
		t.Errorf(`"off" should disable highlighting, got %v, %v`, h, err)
	}
	if h, err := newHighlighter("OFF"); h != nil || err != nil {
		t.Errorf(`"OFF" should also disable highlighting, got %v, %v`, h, err)
	}
	def := mustHighlighter(t, "github")
	unknown := mustHighlighter(t, "no-such-style")
	if string(def.css) != string(unknown.css) {
		t.Errorf("an unknown style should fall back to the default style")
	}
	mono := mustHighlighter(t, "monokai")
	if string(mono.css) == string(def.css) {
		t.Errorf("a different style should produce a different stylesheet")
	}
	if !regexp.MustCompile(`\.chroma\b`).MatchString(string(def.css)) {
		t.Errorf("the stylesheet should be scoped to .chroma")
	}
}

func TestSourcePagesCarryHighlightingOnlyWhereUsed(t *testing.T) {
	repo, dir := newSourceRepo(t)
	writeRepoText(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writeRepoText(t, dir, "notes.txt", "just text\n")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")
	enableSections(t, repo, sectionSource)

	out := generate(t, repo)
	goPage := readOut(t, out, "source/blob/main.go/index.html")
	if !strings.Contains(goPage, `<pre class="blob chroma">`) {
		t.Errorf("a highlighted page should be marked chroma:\n%s", goPage)
	}
	if strings.Count(goPage, "<style>") != 2 || !strings.Contains(goPage, ".chroma") {
		t.Errorf("a highlighted page should carry the token stylesheet after the theme's:\n%.600s", goPage)
	}
	if !strings.Contains(goPage, `<span class="kd">func</span>`) {
		t.Errorf("tokens should be classed on the page")
	}

	// pages that were not highlighted carry no extra stylesheet
	for _, rel := range []string{"source/blob/notes.txt/index.html", "source/index.html", "source/log.html"} {
		page := readOut(t, out, rel)
		if strings.Count(page, "<style>") != 1 || strings.Contains(page, ".chroma") {
			t.Errorf("%s should not carry the token stylesheet", rel)
		}
	}
	if txt := readOut(t, out, "source/blob/notes.txt/index.html"); strings.Contains(txt, "chroma") {
		t.Errorf("a plain text page should not be marked chroma:\n%s", txt)
	}
}

func TestSourceHighlightConfig(t *testing.T) {
	repo, dir := newSourceRepo(t)
	writeRepoText(t, dir, "main.go", "package main\n")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")
	enableSections(t, repo, sectionSource)
	page := func(out string) string { return readOut(t, out, "source/blob/main.go/index.html") }

	// off: plain text, all lines present, no stylesheet
	writeSiteFile(t, repo, "source/highlight", "off\n")
	off := page(generate(t, repo))
	if strings.Contains(off, "chroma") || strings.Count(off, "<style>") != 1 {
		t.Errorf("highlight=off must produce plain pages:\n%.500s", off)
	}
	if !strings.Contains(off, `id="L1"`) {
		t.Errorf("plain pages still have numbered lines")
	}

	// a named style changes the stylesheet
	writeSiteFile(t, repo, "source/highlight", "monokai\n")
	mono := page(generate(t, repo))
	writeSiteFile(t, repo, "source/highlight", "github\n")
	gh := page(generate(t, repo))
	if mono == gh || !strings.Contains(mono, ".chroma") {
		t.Errorf("monokai and github should render different stylesheets")
	}

	// an unknown style neither fails nor changes the default output
	writeSiteFile(t, repo, "source/highlight", "no-such-style\n")
	unknown := page(generate(t, repo))
	if unknown != gh {
		t.Errorf("an unknown style should produce the default (github) output")
	}
}
