package web

import (
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestAutolinkResolvesAndEscapes(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	target, err := repo.Create(core.NewTrackOpts{Title: "Login Flow"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	slug, _ := repo.GetField(target, "slug") // "login-flow"

	// a slug ref that resolves -> relative link to the track page
	got := string(autolink(repo, "../", "see #"+slug+" for details"))
	wantHref := `href="../track/` + target + `.html"`
	if !strings.Contains(got, wantHref) {
		t.Errorf("resolved ref not linked with relative href:\n%s", got)
	}
	if !strings.Contains(got, ">#"+slug+"</a>") {
		t.Errorf("link text should preserve the #ref token:\n%s", got)
	}

	// a dangling ref stays plain text (no link)
	got = string(autolink(repo, "../", "see #no-such-track here"))
	if strings.Contains(got, "<a ") {
		t.Errorf("dangling ref should not be linked:\n%s", got)
	}
	if !strings.Contains(got, "#no-such-track") {
		t.Errorf("dangling ref should remain as text:\n%s", got)
	}

	// HTML is escaped BEFORE linking: a <script> in the body cannot break out
	got = string(autolink(repo, "../", "<script>evil</script> and #"+slug))
	if strings.Contains(got, "<script>") {
		t.Errorf("body HTML must be escaped:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped script tag:\n%s", got)
	}
	// ...and the ref after the escaped content still links
	if !strings.Contains(got, wantHref) {
		t.Errorf("ref should still link after escaped content:\n%s", got)
	}
}

func TestAutolinkCrossRepo(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	url := "https://github.com/major0/other"
	uuid := "0192f1a2-7c3e-7abc-def0-112233445566"
	got := string(autolink(repo, "../", "see #"+url+"?track="+uuid+" upstream"))

	// renders as an external link to the remote repo (not a local track page)
	if !strings.Contains(got, `<a href="`+url+`">`) {
		t.Errorf("cross-repo ref not linked to the remote repo:\n%s", got)
	}
	// label shows repo + short uuid
	if !strings.Contains(got, "#other?track=0192f1a2") {
		t.Errorf("cross-repo label wrong:\n%s", got)
	}
	// it must NOT be treated as a local track link
	if strings.Contains(got, `href="../track/`) {
		t.Errorf("cross-repo ref should not become a local track link:\n%s", got)
	}
}

func TestAutolinkShortId(t *testing.T) {
	dir := t.TempDir()
	repo, _ := core.Init(dir)
	id, _ := repo.Create(core.NewTrackOpts{Title: "x"})
	// a 12-hex short id prefix should resolve
	short := strings.ReplaceAll(id, "-", "")[:12]
	got := string(autolink(repo, "", "ref #"+short))
	if !strings.Contains(got, `href="track/`+id+`.html"`) {
		t.Errorf("short-id ref not linked:\n%s", got)
	}
}
