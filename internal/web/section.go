package web

import (
	"fmt"
	"html/template"

	"github.com/major0/gitrack/internal/core"
)

// Section names (ADR-0037 catalog). Only issues exists in Phase 0; the rest are
// declared so config and nav ordering have stable identifiers as sections land.
const (
	sectionPortal   = "portal"
	sectionBlog     = "blog"
	sectionDocs     = "docs"
	sectionAPI      = "api"
	sectionIssues   = "issues"
	sectionReleases = "releases"
	sectionSource   = "source"
)

// navItem is one entry in the site navigation: the visible label and the
// root-relative href it points to (e.g. "issues.html"). A page prefixes Href
// with its own path-to-root so links work from any subdirectory and file://.
type navItem struct {
	Label string
	Href  string
}

// section renders one part of the site (ADR-0036/0037). Sections are
// independent: each writes its own files under the output dir via the shared
// theme shell, and contributes its nav entries. The interface is intentionally
// small (Go: discover interfaces from usage); it grows only as later sections
// need more.
type section interface {
	// name is the config identifier for the section (e.g. "issues").
	name() string
	// nav returns the navigation entries this section contributes, in order.
	// A section may contribute more than one (issues: tracks, issues, board).
	nav() []navItem
	// render writes the section's pages into the site. It is given the shared
	// context (output dir, repo, resolved nav, theme) so every page gets the
	// same shell and navigation.
	render(sc *siteContext) error
}

// siteContext is the shared state passed to every section's render. It carries
// the output directory, the repo, the resolved site-wide nav (so each section
// renders the same nav bar), and the active theme.
type siteContext struct {
	repo   *core.Repo
	outDir string
	nav    []navItem
	theme  *theme
	// trackPages counts per-track pages written, reported by Generate for
	// backward compatibility with the CLI's "(N track pages)" message.
	trackPages int
}

// buildSections resolves the configured section names to section renderers, in
// config order, dropping unknown names (guardrails-not-enforcement, ADR-0026).
// Only issues is registered in Phase 0; later sections register here as they
// land, so an enabled-but-unbuilt section name is simply skipped until then.
func buildSections(cfg siteConfig) []section {
	out := make([]section, 0, len(cfg.Sections))
	for _, name := range cfg.Sections {
		switch name {
		case sectionIssues:
			out = append(out, &issuesSection{})
		default:
			// Unknown or not-yet-built section: skip (ADR-0026, ADR-0037 §1).
		}
	}
	return out
}

// resolveNav collects the nav entries from all sections, in section order, into
// the single site navigation.
func resolveNav(sections []section) []navItem {
	var nav []navItem
	for _, s := range sections {
		nav = append(nav, s.nav()...)
	}
	return nav
}

// renderPage wraps a section body in the active theme shell and writes it to
// relPath under the output dir, with root as the page's path-to-site-root. The
// nav is taken from the site context so every page shows the same bar.
func (sc *siteContext) renderPage(relPath, root, title string, body template.HTML) error {
	data := pageData{
		Title: title,
		Root:  root,
		Nav:   navWithRoot(sc.nav, root),
		Body:  body,
	}
	if err := sc.theme.render(sc.outDir, relPath, data); err != nil {
		return fmt.Errorf("rendering %s: %w", relPath, err)
	}
	return nil
}

// navWithRoot returns a copy of nav with each Href prefixed by root so links
// resolve from a page living root levels below the site root.
func navWithRoot(nav []navItem, root string) []navItem {
	out := make([]navItem, len(nav))
	for i, n := range nav {
		out[i] = navItem{Label: n.Label, Href: root + n.Href}
	}
	return out
}
