package web

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// docsConfigDir holds the docs section config (ADR-0040):
// .gitrack/config/site/docs/{source,layout}.
const docsConfigDir = siteConfigDir + "/docs"

// defaultDocsSource is the repo-relative docs directory used when no source is
// configured (ADR-0040).
const defaultDocsSource = "docs"

// defaultDocsExcludes are the doc path patterns omitted from the published site
// when no exclude file is authored (ADR-0040): the ADR subtree, which holds
// internal design records not meant for users of the tooling. A project can
// override this with its own exclude file (empty publishes everything).
var defaultDocsExcludes = []string{"adr"}

// docsSection renders the project's Markdown documentation with a configurable
// sidebar layout (ADR-0040). Content is the repo's own docs dir (configurable),
// rendered via renderMarkdown (ADR-0038); output goes under the fixed docs/
// prefix.
type docsSection struct{}

func (s *docsSection) name() string { return sectionDocs }

// nav contributes the single top-bar docs entry, only when the source dir has
// at least one doc (ADR-0040).
func (s *docsSection) nav(sc *siteContext) []navItem {
	_, order, err := s.discover(sc)
	if err != nil || len(order) == 0 {
		return nil
	}
	return []navItem{{Label: "docs", Href: "docs/index.html"}}
}

// render writes every doc page under docs/ plus the docs landing page, each
// carrying the sidebar layout.
func (s *docsSection) render(sc *siteContext) error {
	docs, order, err := s.discover(sc)
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return nil
	}
	sourceDir, err := s.sourceDir(sc)
	if err != nil {
		return err
	}
	layoutSrc, err := readConfigFile(filepath.Join(sc.repo.Root, core.Dir, docsConfigDir, "layout"))
	if err != nil {
		return err
	}
	tree := buildLayout(layoutSrc, docs, order)

	// Render each doc page.
	landingRel := s.landingRel(docs)
	for _, rel := range order {
		src, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(rel)+".md"))
		if err != nil {
			return err
		}
		_, body := splitFrontMatter(string(src))
		relPath := docHref(rel)         // e.g. "docs/adr/0001.html"
		root := rootForDocHref(relPath) // path-to-site-root
		if err := s.writeDocPage(sc, relPath, root, docs[rel], body, tree); err != nil {
			return err
		}
	}

	// Render the landing page at docs/index.html (unless an index.md already
	// mapped there via its rel path).
	if landingRel != "docs" { // docHref("index") == "docs/index.html"
		if err := s.writeLanding(sc, landingRel, sourceDir, docs, tree); err != nil {
			return err
		}
	}
	return nil
}

// writeDocPage renders one doc body plus the sidebar into a docs page.
func (s *docsSection) writeDocPage(sc *siteContext, relPath, root, title, body string, tree []*navNode) error {
	contentHTML, err := renderMarkdown(sc.repo, root, body)
	if err != nil {
		return err
	}
	sidebar := renderSidebar(tree, root, relPath)
	view := docsView{Sidebar: sidebar, Title: title, ContentHTML: contentHTML}
	page, err := renderBody(docsBodyTmpl, view)
	if err != nil {
		return err
	}
	return sc.renderPage(relPath, root, title, page)
}

// writeLanding writes docs/index.html: the source dir's top-level index/README
// rendered if present, else a minimal page that is just the sidebar.
func (s *docsSection) writeLanding(sc *siteContext, landingRel, sourceDir string, docs map[string]string, tree []*navNode) error {
	const relPath = "docs/index.html"
	root := rootForDocHref(relPath)
	title := "Docs"
	body := ""
	if landingRel != "" {
		b, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(landingRel)+".md"))
		if err != nil {
			return err
		}
		_, body = splitFrontMatter(string(b))
		if t, ok := docs[landingRel]; ok {
			title = t
		}
	}
	return s.writeDocPage(sc, relPath, root, title, body, tree)
}

// landingRel returns the rel path to use as the docs landing source: a
// top-level "index" or "README" doc if present, else "". The sentinel value
// "docs" (docHref("index")) tells render() that an index.md already occupies
// docs/index.html, so no separate landing is written.
func (s *docsSection) landingRel(docs map[string]string) string {
	if _, ok := docs["index"]; ok {
		return "docs" // index.md already renders to docs/index.html
	}
	if _, ok := docs["README"]; ok {
		return "README"
	}
	return ""
}

// discover finds every *.md under the source dir and resolves its title. It
// returns a rel->title map and the sorted rel-path order. A missing source dir
// yields empty results (ADR-0040/0026).
func (s *docsSection) discover(sc *siteContext) (map[string]string, []string, error) {
	sourceDir, err := s.sourceDir(sc)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(sourceDir)
	if err != nil || !info.IsDir() {
		return nil, nil, nil //nolint:nilerr // missing source dir => no docs (ADR-0026)
	}
	excludes, err := s.excludes(sc)
	if err != nil {
		return nil, nil, err
	}
	docs := map[string]string{}
	var order []string
	walkErr := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip an entire excluded subtree (e.g. adr/) without descending.
			if path != sourceDir {
				relDir, rerr := filepath.Rel(sourceDir, path)
				if rerr == nil && docExcluded(filepath.ToSlash(relDir), excludes) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		relFS, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		rel := strings.TrimSuffix(filepath.ToSlash(relFS), ".md")
		if docExcluded(rel, excludes) {
			return nil // internal doc: in the repo, not on the site (ADR-0040)
		}
		docs[rel] = pageTitle(path, lastBase(rel))
		order = append(order, rel)
		return nil
	})
	if walkErr != nil {
		return nil, nil, walkErr
	}
	sort.Strings(order)
	return docs, order, nil
}

// excludes returns the doc exclusion patterns: the authored
// .gitrack/config/site/docs/exclude lines, or defaultDocsExcludes when the file
// is absent. An authored-but-empty file publishes everything (ADR-0040).
func (s *docsSection) excludes(sc *siteContext) ([]string, error) {
	path := filepath.Join(sc.repo.Root, core.Dir, docsConfigDir, "exclude")
	raw, err := readConfigFile(path)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return defaultDocsExcludes, nil
	}
	var pats []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pats = append(pats, strings.TrimSuffix(line, "/"))
	}
	return pats, nil
}

// docExcluded reports whether a doc rel path (slash-separated, no extension)
// matches any exclusion pattern: equal to the rel path, a leading path segment
// of it, or a base-name glob (ADR-0040).
func docExcluded(rel string, patterns []string) bool {
	base := lastBase(rel)
	for _, p := range patterns {
		if p == rel {
			return true
		}
		// p is a leading path segment: "adr" matches "adr" and "adr/0001".
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
		// base-name glob, e.g. "*-internal".
		if strings.ContainsAny(p, "*?[") {
			if ok, _ := filepath.Match(p, base); ok {
				return true
			}
		}
	}
	return false
}

// sourceDir returns the absolute docs source directory: the configured path
// (relative to repo root) or the default "docs" (ADR-0040).
func (s *docsSection) sourceDir(sc *siteContext) (string, error) {
	cfg, err := readConfigFile(filepath.Join(sc.repo.Root, core.Dir, docsConfigDir, "source"))
	if err != nil {
		return "", err
	}
	src := defaultDocsSource
	for _, line := range strings.Split(cfg, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		src = line
		break
	}
	return filepath.Join(sc.repo.Root, filepath.FromSlash(src)), nil
}

// lastBase returns the final path element of a slash-separated rel path.
func lastBase(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[i+1:]
	}
	return rel
}

// docsView is the template data for a docs page: the rendered sidebar, the
// page title, and the rendered Markdown content.
type docsView struct {
	Sidebar     template.HTML
	Title       string
	ContentHTML template.HTML
}
