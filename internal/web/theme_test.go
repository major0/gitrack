package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

// writeThemeFile writes a file into the project theme directory name.
func writeThemeFile(t *testing.T, repo *core.Repo, name, rel, content string) {
	t.Helper()
	p := filepath.Join(repo.Root, core.Dir, themeConfigDir, name, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func newThemeRepo(t *testing.T) *core.Repo {
	t.Helper()
	repo, err := core.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return repo
}

func TestLoadThemeFallsBackToDefault(t *testing.T) {
	repo := newThemeRepo(t)
	for _, name := range []string{"", "  ", "nosuch", "../escape", ".hidden", "a/b"} {
		th, err := loadTheme(repo, name)
		if err != nil {
			t.Fatalf("loadTheme(%q): %v", name, err)
		}
		if th.name != defaultThemeName {
			t.Errorf("loadTheme(%q) = %q, want the default", name, th.name)
		}
	}
}

func TestProjectThemeWinsOverBuiltin(t *testing.T) {
	repo := newThemeRepo(t)
	writeThemeFile(t, repo, "default", "theme.css", "/* mine */")
	th, err := loadTheme(repo, "default")
	if err != nil {
		t.Fatalf("loadTheme: %v", err)
	}
	if string(th.files[themeStylesheet]) != "/* mine */" {
		t.Errorf("a project theme must take precedence over a built-in of the same name")
	}
}

func TestProjectThemeNeedsAStylesheet(t *testing.T) {
	repo := newThemeRepo(t)
	writeThemeFile(t, repo, "bare", "theme.js", "//")
	if _, err := loadTheme(repo, "bare"); err == nil || !strings.Contains(err.Error(), themeStylesheet) {
		t.Errorf("a theme directory without %s should be an error naming it, got %v", themeStylesheet, err)
	}
}

func TestProjectThemeSkipsSymlinksAndKeepsLayout(t *testing.T) {
	repo := newThemeRepo(t)
	writeThemeFile(t, repo, "mine", "theme.css", "body{}")
	writeThemeFile(t, repo, "mine", "fonts/a.woff2", "font")
	secret := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(repo.Root, core.Dir, themeConfigDir, "mine", "leak.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	th, err := loadTheme(repo, "mine")
	if err != nil {
		t.Fatalf("loadTheme: %v", err)
	}
	if _, ok := th.files["leak.txt"]; ok {
		t.Errorf("a symlink must not be followed into the site")
	}
	if string(th.files["fonts/a.woff2"]) != "font" {
		t.Errorf("subdirectories keep their layout so the stylesheet can refer to them")
	}
}

func TestProjectThemeIsBounded(t *testing.T) {
	repo := newThemeRepo(t)
	writeThemeFile(t, repo, "big", "theme.css", "body{}")
	writeThemeFile(t, repo, "big", "huge.bin", strings.Repeat("x", maxThemeBytes+1))
	if _, err := loadTheme(repo, "big"); err == nil {
		t.Errorf("a theme over the size cap should be refused")
	}

	many := newThemeRepo(t)
	writeThemeFile(t, many, "many", "theme.css", "body{}")
	for i := 0; i < maxThemeFiles; i++ {
		writeThemeFile(t, many, "many", "f"+strings.Repeat("a", i%7)+string(rune('a'+i%26))+"/"+string(rune('a'+i/26))+".txt", "x")
	}
	if _, err := loadTheme(many, "many"); err == nil {
		t.Errorf("a theme over the file cap should be refused")
	}
}

func TestGenerateWritesThemeAndLinksIt(t *testing.T) {
	repo := newThemeRepo(t)
	if _, err := repo.Create(core.NewTrackOpts{Title: "A track"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	writeSiteFile(t, repo, "theme", "mine\n")
	writeThemeFile(t, repo, "mine", "theme.css", "body{color:red}")
	writeThemeFile(t, repo, "mine", "theme.js", "// mine")
	out := generate(t, repo)

	if got := readOut(t, out, themeCSSAsset); got != "body{color:red}" {
		t.Errorf("theme.css = %q", got)
	}
	if got := readOut(t, out, themeJSAsset); got != "// mine" {
		t.Errorf("theme.js = %q", got)
	}
	page := readOut(t, out, "index.html")
	if !strings.Contains(page, `<script src="assets/theme/theme.js"></script>`) {
		t.Errorf("a theme script is loaded from its file:\n%s", page)
	}
}

func TestPagesAreIdenticalAcrossThemes(t *testing.T) {
	repo := newThemeRepo(t)
	if _, err := repo.Create(core.NewTrackOpts{Title: "A track"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	enableSections(t, repo, sectionIssues, sectionDocs)
	writeRepoFile(t, repo, "docs/guide.md", "# Guide\n\n```go\nfunc main() {}\n```\n")
	writeThemeFile(t, repo, "other", "theme.css", "body{color:blue}")

	a := generate(t, repo)
	writeSiteFile(t, repo, "theme", "other\n")
	b := generate(t, repo)
	for _, rel := range []string{"index.html", "issues.html", "board.html", "docs/guide.html"} {
		if readOut(t, a, rel) != readOut(t, b, rel) {
			t.Errorf("%s differs between themes: the page must not depend on the theme", rel)
		}
	}
	if readOut(t, a, themeCSSAsset) == readOut(t, b, themeCSSAsset) {
		t.Errorf("the themes themselves should differ")
	}
}

func TestGenerateNeverClearsTheAssetsDirectory(t *testing.T) {
	repo := newThemeRepo(t)
	out := filepath.Join(t.TempDir(), "site")
	keep := filepath.Join(out, assetsDir, "mine.txt")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("a file the generator did not write must survive: %v", err)
	}
}

func TestBoardPageLoadsItsScriptAsAFile(t *testing.T) {
	repo := newThemeRepo(t)
	out := generate(t, repo)
	page := readOut(t, out, "board.html")
	if !strings.Contains(page, `<script src="assets/board.js"></script>`) || strings.Contains(page, "import ") {
		t.Errorf("the board loads assets/board.js and carries no inline script:\n%s", page)
	}
	if !strings.Contains(readOut(t, out, boardAsset), "import(") {
		t.Errorf("board.js should load Mermaid with a dynamic import so it works from file://")
	}
	if strings.Contains(readOut(t, out, "index.html"), "board.js") {
		t.Errorf("only the board page loads the board script")
	}
}

// Every built-in theme can be selected by name, and choosing one changes the
// stylesheet and nothing in any page.
func TestBuiltinThemesSwapWithoutChangingPages(t *testing.T) {
	entries, err := builtinThemes.ReadDir("themes")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least the default and one other built-in theme, got %d", len(entries))
	}
	repo := pureSiteRepo(t)
	base := generate(t, repo)
	basePages := allPages(t, base)
	baseCSS := readOut(t, base, themeCSSAsset)
	for _, e := range entries {
		name := e.Name()
		if name == defaultThemeName {
			continue
		}
		writeSiteFile(t, repo, "theme", name+"\n")
		out := generate(t, repo)
		if readOut(t, out, themeCSSAsset) == baseCSS {
			t.Errorf("theme %q should have its own stylesheet", name)
		}
		pages := allPages(t, out)
		if len(pages) != len(basePages) {
			t.Errorf("theme %q: %d pages, default has %d", name, len(pages), len(basePages))
		}
		for rel, page := range basePages {
			if pages[rel] != page {
				t.Errorf("theme %q: %s differs from the default theme's page", name, rel)
			}
		}
	}
}
