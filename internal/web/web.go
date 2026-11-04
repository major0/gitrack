// Package web generates a self-contained static site from the repository and
// its tracker, to the constraints of ADR-0029: the output is a plain directory
// of HTML that renders from file:// or any dumb host with no running server.
//
// The site is built from independent SECTIONS over a shared THEME (ADR-0036/
// 0037). Each section renders part of the site (issues, and in future docs,
// source, releases, portal, blog, api) into the theme's shell; a simple config
// under .gitrack/config/site/ selects which sections are enabled and the theme.
// With no config the issues section is the only one enabled with the built-in
// theme, so the generated site matches pre-ADR-0037 output exactly.
//
// Pages carry content and structure only. Every stylesheet and script is a file
// under assets/, chosen by the theme, so the HTML is the same whichever theme is
// used (ADR-0048). The site needs no network; the dependency digraph is a
// progressive enhancement (Mermaid via CDN) above an always-present HTML
// fallback.
package web

import (
	"html/template"

	"github.com/major0/gitrack/internal/core"
)

// pageData is the common template data for the theme shell: the site-relative
// path prefix back to root (so links work from any subdirectory and file://),
// the resolved navigation (each Href already root-prefixed), and the body.
type pageData struct {
	Title     string
	Root      string // relative path to site root, e.g. "" or "../"
	Section   string // the section the page belongs to, stamped as data-section
	Nav       []navItem
	Body      template.HTML
	Highlight bool     // link assets/highlight.css
	Scripts   []string // root-relative script files, loaded at the end of the body
}

// Generate writes the static site for repo into outDir, overwriting prior
// output. It loads the site config, builds the enabled sections over the
// resolved theme, renders each, and returns the number of per-track pages
// written (preserving the CLI's "(N track pages)" report).
func Generate(repo *core.Repo, outDir string) (int, error) {
	cfg, err := loadSiteConfig(repo)
	if err != nil {
		return 0, err
	}
	th, err := loadTheme(repo, cfg.Theme)
	if err != nil {
		return 0, err
	}
	sections := buildSections(cfg)
	sc := &siteContext{
		repo:        repo,
		outDir:      outDir,
		theme:       th,
		homeSection: homeSectionName(sections, repo),
		blogEnabled: hasSection(sections, sectionBlog),

		issuesEnabled:   hasSection(sections, sectionIssues),
		releasesEnabled: hasSection(sections, sectionReleases),
		sourceEnabled:   hasSection(sections, sectionSource),
	}
	// Nav depends on home-ownership and sibling sections, so resolve it after
	// the context is populated (ADR-0039).
	sc.nav = resolveNav(sections, sc)
	for _, s := range sections {
		sc.section = s.name()
		if err := s.render(sc); err != nil {
			return 0, err
		}
	}
	if err := sc.writeAssets(); err != nil {
		return 0, err
	}
	return sc.trackPages, nil
}

// short truncates an id to its first 8 characters for display.
func short(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}
