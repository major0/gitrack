package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestLoadSiteConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	cfg, err := loadSiteConfig(repo)
	if err != nil {
		t.Fatalf("loadSiteConfig: %v", err)
	}
	if len(cfg.Sections) != 1 || cfg.Sections[0] != sectionIssues {
		t.Errorf("default sections = %v, want [issues]", cfg.Sections)
	}
	if cfg.Theme != "" {
		t.Errorf("default theme = %q, want empty (built-in)", cfg.Theme)
	}
}

func TestLoadSiteConfigReadsFiles(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	base := filepath.Join(repo.Root, core.Dir, siteConfigDir)
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// comments and blank lines are ignored; order is preserved
	if err := os.WriteFile(filepath.Join(base, "sections"),
		[]byte("# sections\nissues\n\ndocs\n"), 0o644); err != nil {
		t.Fatalf("write sections: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "theme"),
		[]byte("midnight\n"), 0o644); err != nil {
		t.Fatalf("write theme: %v", err)
	}
	cfg, err := loadSiteConfig(repo)
	if err != nil {
		t.Fatalf("loadSiteConfig: %v", err)
	}
	if strings.Join(cfg.Sections, ",") != "issues,docs" {
		t.Errorf("sections = %v, want [issues docs]", cfg.Sections)
	}
	if cfg.Theme != "midnight" {
		t.Errorf("theme = %q, want midnight", cfg.Theme)
	}
}

func TestBuildSectionsDropsUnknown(t *testing.T) {
	// Only issues is registered in Phase 0; an unbuilt/unknown name is skipped
	// (guardrails, ADR-0026).
	got := buildSections(siteConfig{Sections: []string{"issues", "docs", "bogus"}})
	if len(got) != 1 {
		t.Fatalf("buildSections kept %d sections, want 1 (issues only)", len(got))
	}
	if got[0].name() != sectionIssues {
		t.Errorf("kept section %q, want issues", got[0].name())
	}
}

func TestDisabledSectionLeavesNoNav(t *testing.T) {
	// With no sections enabled, the nav is empty and no section files exist.
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	base := filepath.Join(repo.Root, core.Dir, siteConfigDir)
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// an enabled-but-unbuilt section name yields zero built sections, so the
	// site has no nav and no index/issues/board pages.
	if err := os.WriteFile(filepath.Join(base, "sections"),
		[]byte("docs\n"), 0o644); err != nil {
		t.Fatalf("write sections: %v", err)
	}
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); !os.IsNotExist(err) {
		t.Errorf("index.html should not exist when issues section is disabled (err=%v)", err)
	}
}

func TestNavWithRoot(t *testing.T) {
	nav := []navItem{{Label: "issues", Href: "issues.html"}}
	got := navWithRoot(nav, "../")
	if got[0].Href != "../issues.html" {
		t.Errorf("Href = %q, want ../issues.html", got[0].Href)
	}
	// original is not mutated
	if nav[0].Href != "issues.html" {
		t.Errorf("source nav mutated: %q", nav[0].Href)
	}
}
