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
	// It is given the site context so entries can depend on home-ownership and
	// which sibling sections are enabled (ADR-0039).
	nav(sc *siteContext) []navItem
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
	// homeSection is the name of the section that owns the site root index.html
	// (the first enabled section, ADR-0039). A section compares its own name to
	// decide whether it writes index.html or its secondary path.
	homeSection string
	// blogEnabled reports whether the blog section is enabled, so the portal
	// section can include the blog link in its nav (ADR-0039).
	blogEnabled bool
	// issuesEnabled reports whether the issues section is enabled. Other sections
	// link to track pages only when those pages will exist (ADR-0041).
	issuesEnabled bool
	// sourceEnabled reports whether the source section is enabled, so track pages
	// can link a mentioning commit to its source page (ADR-0042).
	sourceEnabled bool
	// srcMemo caches the source snapshot and log for this generation, so the
	// source section and the track pages agree on which commit pages exist.
	srcMemo sourceMemo
	// releasesEnabled reports whether the releases section is enabled, so track
	// pages can link "shipped in" to the release page (ADR-0041).
	releasesEnabled bool
	// trackPages counts per-track pages written, reported by Generate for
	// backward compatibility with the CLI's "(N track pages)" message.
	trackPages int
}

// isHome reports whether the named section owns the site root index.html.
func (sc *siteContext) isHome(name string) bool { return sc.homeSection == name }

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
		case sectionPortal:
			out = append(out, &portalSection{})
		case sectionBlog:
			out = append(out, &blogSection{})
		case sectionDocs:
			out = append(out, &docsSection{})
		case sectionReleases:
			out = append(out, &releasesSection{})
		case sectionSource:
			out = append(out, &sourceSection{})
		default:
			// Unknown or not-yet-built section: skip (ADR-0026, ADR-0037 §1).
		}
	}
	return out
}

// resolveNav collects the nav entries from all sections, in section order, into
// the single site navigation. It is given the context so section nav can depend
// on home-ownership and sibling sections (ADR-0039).
func resolveNav(sections []section, sc *siteContext) []navItem {
	var nav []navItem
	for _, s := range sections {
		nav = append(nav, s.nav(sc)...)
	}
	return nav
}

// homeProvider is an optional capability: a section implements it to report
// whether it can actually produce the site root index.html for this repo. A
// section without a home page (e.g. portal with no front.md) declines, so home
// falls through to the next section (ADR-0039).
type homeProvider interface {
	providesHome(repo *core.Repo) bool
}

// homeSectionName returns the name of the first enabled section that can own the
// site root index.html (ADR-0039). A section that does not implement
// homeProvider is assumed to always provide a home (e.g. issues). Returns "" when
// no enabled section can provide one.
func homeSectionName(sections []section, repo *core.Repo) string {
	for _, s := range sections {
		if hp, ok := s.(homeProvider); ok {
			if !hp.providesHome(repo) {
				continue
			}
		}
		return s.name()
	}
	return ""
}

// hasSection reports whether a section with the given name is in the set.
func hasSection(sections []section, name string) bool {
	for _, s := range sections {
		if s.name() == name {
			return true
		}
	}
	return false
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
