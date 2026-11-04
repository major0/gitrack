package web

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// blogContentDir is the tracker-relative directory holding blog posts
// (ADR-0039): .gitrack/config/site/blog/*.md.
const blogContentDir = siteConfigDir + "/blog"

// blogSection renders the news blog: one page per post under blog/, plus a
// blog/index.html listing posts newest-first (ADR-0039). Posts are authored
// repo Markdown rendered via renderMarkdown (ADR-0038).
type blogSection struct{}

func (s *blogSection) name() string { return sectionBlog }

// post is one blog entry: its slug (file base), title, date (YYYY-MM-DD or ""),
// output href, and source path.
type post struct {
	slug  string
	title string
	date  string
	href  string // root-relative, e.g. "blog/first.html"
	src   string
}

// nav contributes a blog entry only when the portal section is NOT also
// enabled (portal renders the blog link itself for a single coherent bar,
// ADR-0039). Missing posts yield no entry.
func (s *blogSection) nav(sc *siteContext) []navItem {
	if sc.homeSection == sectionPortal {
		return nil // portal owns the blog nav link
	}
	posts, err := s.posts(sc)
	if err != nil || len(posts) == 0 {
		return nil
	}
	return []navItem{{Label: "blog", Href: "blog/index.html"}}
}

// render writes each post page and the blog index.
func (s *blogSection) render(sc *siteContext) error {
	posts, err := s.posts(sc)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return nil
	}
	for _, p := range posts {
		src, err := os.ReadFile(p.src)
		if err != nil {
			return err
		}
		_, body := splitFrontMatter(string(src))
		md, err := sc.markdown("../", body)
		if err != nil {
			return err
		}
		view := postView{Title: p.title, Date: p.date, BodyHTML: md.HTML}
		rendered, err := renderBody(postBodyTmpl, view)
		if err != nil {
			return err
		}
		if err := sc.renderPageExtra("blog/"+p.slug+".html", "../", p.title, rendered, sc.extraFor(md.CSS, md.HasCode)); err != nil {
			return err
		}
	}
	return s.writeIndex(sc, posts)
}

// writeIndex renders blog/index.html listing posts newest-first.
func (s *blogSection) writeIndex(sc *siteContext, posts []post) error {
	rows := make([]postRow, len(posts))
	for i, p := range posts {
		rows[i] = postRow{Slug: p.slug, Title: p.title, Date: p.date}
	}
	body, err := renderBody(blogIndexTmpl, rows)
	if err != nil {
		return err
	}
	return sc.renderPage("blog/index.html", "../", "Blog", body)
}

// postRow is one row in the blog index (exported for html/template access).
type postRow struct {
	Slug  string
	Title string
	Date  string
}

// posts lists blog posts sorted by date descending (newest first); undated
// posts sort last by slug (ADR-0039). A missing content dir yields nil.
func (s *blogSection) posts(sc *siteContext) ([]post, error) {
	dir := filepath.Join(sc.repo.Root, core.Dir, blogContentDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var posts []post
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		src := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(src)
		if err != nil {
			return nil, err
		}
		fm, body := splitFrontMatter(string(b))
		title := fm.Title
		if title == "" {
			if m := h1Re.FindStringSubmatch(body); m != nil {
				title = strings.TrimSpace(m[1])
			} else {
				title = slug
			}
		}
		posts = append(posts, post{
			slug:  slug,
			title: title,
			date:  fm.Date,
			href:  "blog/" + slug + ".html",
			src:   src,
		})
	}
	sort.Slice(posts, func(i, j int) bool {
		// dated posts before undated; among dated, newer (greater date string)
		// first; ties and undated by slug ascending. ISO dates sort
		// lexicographically, so string comparison is chronological.
		di, dj := posts[i].date, posts[j].date
		if (di == "") != (dj == "") {
			return di != "" // dated first
		}
		if di != dj {
			return di > dj // newer first
		}
		return posts[i].slug < posts[j].slug
	})
	return posts, nil
}

// postView is the template data for a single post page: the rendered Markdown
// body wrapped with the post's title and date.
type postView struct {
	Title    string
	Date     string
	BodyHTML template.HTML
}
