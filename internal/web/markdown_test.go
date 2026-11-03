package web

import (
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestRenderMarkdownBasics(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	src := "# Title\n\nSome *text* with a list:\n\n- one\n- two\n\n```\ncode block\n```\n\n[a link](https://example.com)\n"
	got, err := renderMarkdown(repo, "", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	s := string(got)
	for _, want := range []string{"<h1", "Title", "<ul>", "<li>one</li>", "<pre><code>", "code block", `href="https://example.com"`} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown output missing %q:\n%s", want, s)
		}
	}
	// no external stylesheet / script injected by the renderer itself
	if strings.Contains(s, "<link") || strings.Contains(s, "<script") {
		t.Errorf("markdown output must not add external assets:\n%s", s)
	}
}

func TestRenderMarkdownAutolinksProse(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(core.NewTrackOpts{Title: "Login Flow"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	slug, _ := repo.GetField(id, "slug") // "login-flow"

	src := "See #" + slug + " for details.\n"
	got, err := renderMarkdown(repo, "../", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	s := string(got)
	wantHref := `href="../track/` + id + `.html"`
	if !strings.Contains(s, wantHref) {
		t.Errorf("prose #ref not linked with relative href:\n%s", s)
	}
	if !strings.Contains(s, ">#"+slug+"</a>") {
		t.Errorf("link text should preserve the #ref token:\n%s", s)
	}
}

func TestRenderMarkdownSkipsCode(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, _ := repo.Create(core.NewTrackOpts{Title: "Login Flow"})
	slug, _ := repo.GetField(id, "slug")

	// same resolvable token inside an inline code span and a fenced block
	src := "inline `#" + slug + "` and\n\n```\n#" + slug + "\n```\n"
	got, err := renderMarkdown(repo, "", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	s := string(got)
	if strings.Contains(s, "<a ") {
		t.Errorf("a #ref inside code must NOT be linked:\n%s", s)
	}
	// the token text survives as code content
	if !strings.Contains(s, "#"+slug) {
		t.Errorf("code content should retain the token:\n%s", s)
	}
}

func TestRenderMarkdownCrossRepo(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	url := "https://github.com/major0/other"
	uuid := "0192f1a2-7c3e-7abc-def0-112233445566"
	src := "see #" + url + "?track=" + uuid + " upstream\n"
	got, err := renderMarkdown(repo, "../", src)
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, `href="`+url+`"`) {
		t.Errorf("cross-repo ref not linked to the remote repo:\n%s", s)
	}
	if !strings.Contains(s, "#other?track=0192f1a2") {
		t.Errorf("cross-repo label wrong:\n%s", s)
	}
	if strings.Contains(s, `href="../track/`) {
		t.Errorf("cross-repo ref should not become a local track link:\n%s", s)
	}
}

func TestRenderMarkdownDanglingRefStaysText(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, err := renderMarkdown(repo, "", "see #no-such-track here\n")
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	s := string(got)
	if strings.Contains(s, "<a ") {
		t.Errorf("dangling ref should not be linked:\n%s", s)
	}
	if !strings.Contains(s, "#no-such-track") {
		t.Errorf("dangling ref should remain as text:\n%s", s)
	}
}

func TestRenderMarkdownEscapesRawHTML(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, err := renderMarkdown(repo, "", "before <script>alert(1)</script> after\n")
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	s := string(got)
	// goldmark's default disables raw HTML: an embedded <script> is NOT emitted
	// as live markup (it is replaced with an omitted-HTML comment), so authored
	// content cannot inject script/iframe (ADR-0038 Safety).
	if strings.Contains(s, "<script>") || strings.Contains(s, "</script>") {
		t.Errorf("raw HTML must not be rendered as live markup:\n%s", s)
	}
	// surrounding prose text is preserved
	if !strings.Contains(s, "before") || !strings.Contains(s, "after") {
		t.Errorf("prose around raw HTML should survive:\n%s", s)
	}
}
