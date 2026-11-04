package web

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// portalContentDir is the tracker-relative directory holding portal pages
// (ADR-0039): .gitrack/config/site/portal/*.md.
const portalContentDir = siteConfigDir + "/portal"

// frontPageFile is the portal page that becomes the site root index.html when
// the portal section owns the home page (ADR-0039).
const frontPageFile = "front.md"

// h1Re matches the first ATX H1 line, used as a title fallback (ADR-0039).
var h1Re = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)

// portalSection renders the marketing pages: a front page plus one page per
// Markdown file under the portal content dir (about, contact, ...). Content is
// authored repo Markdown rendered via renderMarkdown (ADR-0038/0039).
type portalSection struct{}

func (s *portalSection) name() string { return sectionPortal }

// providesHome reports whether the portal can own the site root: only when a
// front.md exists. Otherwise home falls through to the next section (ADR-0039).
func (s *portalSection) providesHome(repo *core.Repo) bool {
	p := filepath.Join(repo.Root, core.Dir, portalContentDir, frontPageFile)
	_, err := os.Stat(p)
	return err == nil
}

// page is one portal Markdown file: its source path, the output-relative href,
// the resolved title, and whether it is the front page.
type portalPage struct {
	base  string // file base name without extension, e.g. "about"
	title string
	href  string // root-relative output path, e.g. "portal/about.html"
	src   string // absolute source path
	front bool
}

// nav contributes the portal's entries: home (only when portal owns the root),
// each non-front page, then the blog link when the blog section is enabled
// (ADR-0039). Missing content yields no entries.
func (s *portalSection) nav(sc *siteContext) []navItem {
	pages, err := s.pages(sc)
	if err != nil {
		return nil
	}
	var nav []navItem
	if sc.isHome(sectionPortal) {
		nav = append(nav, navItem{Label: "home", Href: "index.html"})
	}
	for _, p := range pages {
		if p.front {
			continue
		}
		nav = append(nav, navItem{Label: p.title, Href: p.href})
	}
	if sc.blogEnabled {
		nav = append(nav, navItem{Label: "blog", Href: "blog/index.html"})
	}
	return nav
}

// render writes the front page (to index.html when portal is home) and each
// other portal page under portal/.
func (s *portalSection) render(sc *siteContext) error {
	pages, err := s.pages(sc)
	if err != nil {
		return err
	}
	for _, p := range pages {
		src, err := os.ReadFile(p.src)
		if err != nil {
			return err
		}
		_, body := splitFrontMatter(string(src))
		relPath, root := s.output(sc, p)
		md, err := sc.markdown(root, body)
		if err != nil {
			return err
		}
		if err := sc.renderPageCSS(relPath, root, p.title, md.HTML, md.CSS); err != nil {
			return err
		}
	}
	return nil
}

// output returns the output-relative path and path-to-root for a page: the
// front page goes to index.html (root "") when portal is home; other pages go
// to portal/<base>.html (root "../").
func (s *portalSection) output(sc *siteContext, p portalPage) (relPath, root string) {
	if p.front && sc.isHome(sectionPortal) {
		return "index.html", ""
	}
	return "portal/" + p.base + ".html", "../"
}

// pages lists the portal pages in deterministic order: the front page first (if
// present), then the rest by base name. A missing content dir yields nil.
func (s *portalSection) pages(sc *siteContext) ([]portalPage, error) {
	dir := filepath.Join(sc.repo.Root, core.Dir, portalContentDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var pages []portalPage
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".md")
		src := filepath.Join(dir, e.Name())
		title := pageTitle(src, base)
		front := e.Name() == frontPageFile
		href := "portal/" + base + ".html"
		if front {
			href = "index.html"
		}
		pages = append(pages, portalPage{base: base, title: title, href: href, src: src, front: front})
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].front != pages[j].front {
			return pages[i].front // front first
		}
		return pages[i].base < pages[j].base
	})
	return pages, nil
}

// pageTitle resolves a page/post title: front-matter title, else the first H1,
// else the file base name (ADR-0039).
func pageTitle(src, base string) string {
	b, err := os.ReadFile(src)
	if err != nil {
		return base
	}
	fm, body := splitFrontMatter(string(b))
	if fm.Title != "" {
		return fm.Title
	}
	if m := h1Re.FindStringSubmatch(body); m != nil {
		return strings.TrimSpace(m[1])
	}
	return base
}
