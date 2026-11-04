package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/major0/gitrack/internal/core"
	"github.com/major0/gitrack/internal/gitsrc"
)

// sourceConfigDir holds the source section config (ADR-0042):
// .gitrack/config/site/source/{ref,commits}.
const sourceConfigDir = siteConfigDir + "/source"

const (
	// defaultSourceCommits is how many log entries are rendered when the
	// "commits" config is absent or invalid.
	defaultSourceCommits = 100
	// maxBlobBytes caps how much of one file is rendered (ADR-0042).
	maxBlobBytes = 512 << 10
	// maxPatchBytes caps how much of one commit's patch is rendered.
	maxPatchBytes = 256 << 10
)

// sourceSection renders a gitweb-style browser: the tree at one ref, the recent
// log, and a page per recent commit (ADR-0042). History is bounded by the log
// size, not by the repository's length.
type sourceSection struct{}

func (s *sourceSection) name() string { return sectionSource }

// sourceSnapshot is the resolved view the section renders.
type sourceSnapshot struct {
	git     *gitsrc.Repo
	ref     string
	sha     string
	commits int
}

// snapshot resolves the configured ref. It returns nil, nil when there is
// nothing to browse (not a git repository, no commits, or an unresolvable ref),
// so the section emits nothing (ADR-0026).
func (s *sourceSection) snapshot(sc *siteContext) (*sourceSnapshot, error) {
	base := filepath.Join(sc.repo.Root, core.Dir, sourceConfigDir)
	ref, err := readConfigValue(filepath.Join(base, "ref"), "HEAD")
	if err != nil {
		return nil, err
	}
	n := defaultSourceCommits
	raw, err := readConfigValue(filepath.Join(base, "commits"), "")
	if err != nil {
		return nil, err
	}
	if parsed, perr := strconv.Atoi(raw); perr == nil && parsed > 0 {
		n = parsed // a missing or malformed value keeps the default
	}
	g := gitsrc.Open(sc.repo.Root)
	sha, err := g.ResolveCommit(ref)
	if errors.Is(err, gitsrc.ErrNoCommit) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sourceSnapshot{git: g, ref: ref, sha: sha, commits: n}, nil
}

// nav contributes one source entry, only when there is something to browse.
func (s *sourceSection) nav(sc *siteContext) []navItem {
	snap, err := s.snapshot(sc)
	if err != nil || snap == nil {
		return nil
	}
	return []navItem{{Label: "source", Href: treePage("").href}}
}

// render writes the tree and blob pages for the snapshot, then the log and a
// page per logged commit.
func (s *sourceSection) render(sc *siteContext) error {
	snap, err := s.snapshot(sc)
	if err != nil || snap == nil {
		return err
	}
	files, err := snap.git.Files(snap.sha)
	if err != nil {
		return err
	}
	commits, err := snap.git.Log(snap.sha, snap.commits)
	if err != nil {
		return err
	}
	dirs, valid := indexTree(files)
	for dir, node := range dirs {
		if err := s.writeTree(sc, snap, commits, dir, node); err != nil {
			return err
		}
	}
	if err := s.writeBlobs(sc, snap, valid); err != nil {
		return err
	}
	return s.writeHistory(sc, snap, commits)
}

// ---- addressing ------------------------------------------------------------

// srcPage names one output page twice: file is the on-disk path (raw names), and
// href is the same location as a URL (each segment escaped). They differ only for
// names containing characters such as space, '#', or '%'.
type srcPage struct{ file, href string }

// root is the path-to-site-root prefix for links written on this page.
func (p srcPage) root() string { return rootForHref(p.file) }

// escapePath URL-escapes each segment of a slash-separated path.
func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

// treePage is the page for a directory; the root directory is source/index.html.
func treePage(dir string) srcPage {
	if dir == "" {
		return srcPage{"source/index.html", "source/index.html"}
	}
	return srcPage{
		file: "source/tree/" + dir + "/index.html",
		href: "source/tree/" + escapePath(dir) + "/index.html",
	}
}

// blobPage is the page for a file. Every page is a directory index, so a file
// can never collide with a directory of the same name (ADR-0042).
func blobPage(path string) srcPage {
	return srcPage{
		file: "source/blob/" + path + "/index.html",
		href: "source/blob/" + escapePath(path) + "/index.html",
	}
}

func commitPage(sha string) srcPage {
	f := "source/commit/" + sha + ".html"
	return srcPage{f, f}
}

var logPage = srcPage{"source/log.html", "source/log.html"}

// crumb is one breadcrumb; an empty Href marks the current (last) one.
type crumb struct{ Name, Href string }

// breadcrumbs links every ancestor of path back to its tree page and leaves the
// leaf plain. root prefixes the links for the page they appear on.
func breadcrumbs(root, path string) []crumb {
	if path == "" {
		return []crumb{{Name: "source"}}
	}
	parts := strings.Split(path, "/")
	cs := []crumb{{Name: "source", Href: root + treePage("").href}}
	for i := 0; i < len(parts)-1; i++ {
		sub := strings.Join(parts[:i+1], "/")
		cs = append(cs, crumb{Name: parts[i], Href: root + treePage(sub).href})
	}
	return append(cs, crumb{Name: parts[len(parts)-1]})
}

// ---- tree ------------------------------------------------------------------

// dirNode is one directory in the snapshot: its subdirectory names and files.
type dirNode struct {
	subdirs map[string]bool
	files   []gitsrc.Entry
}

// indexTree groups entries by directory, returning the directories keyed by path
// ("" is the root) and the entries that were accepted. An entry whose path has an
// empty, "." or ".." component is skipped, so a malformed tree can never write
// outside the output directory (ADR-0042).
func indexTree(files []gitsrc.Entry) (map[string]*dirNode, []gitsrc.Entry) {
	dirs := map[string]*dirNode{"": {subdirs: map[string]bool{}}}
	node := func(p string) *dirNode {
		n, ok := dirs[p]
		if !ok {
			n = &dirNode{subdirs: map[string]bool{}}
			dirs[p] = n
		}
		return n
	}
	var valid []gitsrc.Entry
	for _, e := range files {
		parts := strings.Split(e.Path, "/")
		if !safeParts(parts) {
			continue
		}
		parent := ""
		for _, name := range parts[:len(parts)-1] {
			dir := name
			if parent != "" {
				dir = parent + "/" + name
			}
			node(parent).subdirs[name] = true
			node(dir)
			parent = dir
		}
		n := node(parent)
		n.files = append(n.files, e)
		valid = append(valid, e)
	}
	return dirs, valid
}

// safeParts reports whether every path component is usable as a file name.
func safeParts(parts []string) bool {
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}

// treeRow is one line of a directory listing.
type treeRow struct {
	Name string
	Href string // empty for a submodule, which has no page
	Note string
	Size string
}

// treeView is the template data for a directory page.
type treeView struct {
	Crumbs     []crumb
	Ref        string
	Short      string
	CommitHref string
	LogHref    string
	Rows       []treeRow
}

func (s *sourceSection) writeTree(sc *siteContext, snap *sourceSnapshot, commits []gitsrc.Commit, dir string, n *dirNode) error {
	page := treePage(dir)
	root := page.root()

	names := make([]string, 0, len(n.subdirs))
	for name := range n.subdirs {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]treeRow, 0, len(names)+len(n.files))
	for _, name := range names {
		sub := name
		if dir != "" {
			sub = dir + "/" + name
		}
		rows = append(rows, treeRow{Name: name + "/", Href: root + treePage(sub).href})
	}
	sort.Slice(n.files, func(i, j int) bool { return n.files[i].Path < n.files[j].Path })
	for _, e := range n.files {
		row := treeRow{Name: e.Path[strings.LastIndex(e.Path, "/")+1:]}
		switch {
		case e.IsSubmodule():
			row.Note = "submodule @ " + shortSHA(e.SHA)
		case e.IsSymlink():
			row.Href = root + blobPage(e.Path).href
			row.Note = "symlink"
		default:
			row.Href = root + blobPage(e.Path).href
			row.Size = humanSize(e.Size)
		}
		rows = append(rows, row)
	}

	view := treeView{
		Crumbs:  breadcrumbs(root, dir),
		Ref:     snap.ref,
		Short:   shortSHA(snap.sha),
		LogHref: root + logPage.href,
		Rows:    rows,
	}
	if len(commits) > 0 {
		view.CommitHref = root + commitPage(commits[0].SHA).href
	}
	body, err := renderBody(treeBodyTmpl, view)
	if err != nil {
		return err
	}
	title := "source"
	if dir != "" {
		title = dir + " - source"
	}
	return sc.renderPage(page.file, root, title, body)
}

// ---- blobs -----------------------------------------------------------------

// blobView is the template data for a file page.
type blobView struct {
	Crumbs   []crumb
	Size     string
	Notes    []string
	CodeHTML template.HTML
}

func (s *sourceSection) writeBlobs(sc *siteContext, snap *sourceSnapshot, files []gitsrc.Entry) error {
	blobs, err := snap.git.OpenBlobs()
	if err != nil {
		return err
	}
	defer func() { _ = blobs.Close() }()
	for _, e := range files {
		if e.IsSubmodule() {
			continue // listed in its directory, no page
		}
		blob, err := blobs.Read(e.SHA, maxBlobBytes)
		if err != nil {
			return fmt.Errorf("reading %s: %w", e.Path, err)
		}
		page := blobPage(e.Path)
		root := page.root()
		view := blobView{Crumbs: breadcrumbs(root, e.Path), Size: humanSize(blob.Size)}
		switch {
		case e.IsSymlink():
			view.Notes = append(view.Notes, "symbolic link to "+strings.TrimSpace(string(blob.Data)))
		case blob.Binary:
			view.Notes = append(view.Notes, "binary file")
		case blob.Size == 0:
			view.Notes = append(view.Notes, "empty file")
		default:
			if blob.Truncated {
				view.Notes = append(view.Notes, fmt.Sprintf("showing the first %s of %s",
					humanSize(int64(len(blob.Data))), humanSize(blob.Size)))
			}
			view.CodeHTML = renderCode(blob.Data)
		}
		body, err := renderBody(blobBodyTmpl, view)
		if err != nil {
			return err
		}
		if err := sc.renderPage(page.file, root, e.Path+" - source", body); err != nil {
			return err
		}
	}
	return nil
}

// renderCode escapes file content into numbered lines. Each line is a span with
// an id (L1, L2, ...) so #L12 links work; the visible number comes from a CSS
// counter, so selecting and copying code does not pick up the numbers.
func renderCode(data []byte) template.HTML {
	text := strings.ToValidUTF8(string(data), "\uFFFD")
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1] // a trailing newline does not start another line
	}
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, `<span class="l" id="L%d">%s</span>`+"\n", i+1, template.HTMLEscapeString(line))
	}
	return template.HTML(b.String()) //nolint:gosec // every line is HTML-escaped above
}

// ---- history ---------------------------------------------------------------

// logRow is one line of the commit log.
type logRow struct {
	Href    string
	Subject string
	Short   string
	Author  string
	Date    string
}

// logView is the template data for the log page.
type logView struct {
	SourceHref string
	Rows       []logRow
}

// parentRef is a link to a parent commit, or plain text when its page was not
// rendered because it falls outside the log.
type parentRef struct{ Short, Href string }

// changeRow is one file in a commit's change summary.
type changeRow struct {
	Path           string
	Added, Deleted int
	Binary         bool
}

// commitView is the template data for a commit page.
type commitView struct {
	SourceHref  string
	LogHref     string
	Short       string
	SHA         string
	SubjectHTML template.HTML
	BodyHTML    template.HTML
	Author      string
	Date        string
	Parents     []parentRef
	Changes     []changeRow
	DiffHTML    template.HTML
	Truncated   bool
}

func (s *sourceSection) writeHistory(sc *siteContext, snap *sourceSnapshot, commits []gitsrc.Commit) error {
	inLog := make(map[string]bool, len(commits))
	for _, c := range commits {
		inLog[c.SHA] = true
	}

	logRoot := logPage.root()
	rows := make([]logRow, len(commits))
	for i, c := range commits {
		rows[i] = logRow{
			Href:    logRoot + commitPage(c.SHA).href,
			Subject: subjectOrPlaceholder(c.Subject),
			Short:   shortSHA(c.SHA),
			Author:  c.Author,
			Date:    formatCommitDate(c),
		}
	}
	body, err := renderBody(logBodyTmpl, logView{SourceHref: logRoot + treePage("").href, Rows: rows})
	if err != nil {
		return err
	}
	if err := sc.renderPage(logPage.file, logRoot, "log - source", body); err != nil {
		return err
	}

	for _, c := range commits {
		if err := s.writeCommit(sc, snap, c, inLog); err != nil {
			return err
		}
	}
	return nil
}

func (s *sourceSection) writeCommit(sc *siteContext, snap *sourceSnapshot, c gitsrc.Commit, inLog map[string]bool) error {
	page := commitPage(c.SHA)
	root := page.root()
	detail, err := snap.git.Describe(c, maxPatchBytes)
	if err != nil {
		return fmt.Errorf("describing %s: %w", shortSHA(c.SHA), err)
	}
	view := commitView{
		SourceHref:  root + treePage("").href,
		LogHref:     root + logPage.href,
		Short:       shortSHA(c.SHA),
		SHA:         c.SHA,
		SubjectHTML: autolink(sc.repo, root, subjectOrPlaceholder(c.Subject)),
		Author:      c.Author,
		Date:        formatCommitDate(c),
		DiffHTML:    renderDiff(detail.Patch),
		Truncated:   detail.PatchTruncated,
	}
	if c.Body != "" {
		view.BodyHTML = autolink(sc.repo, root, c.Body)
	}
	for _, p := range c.Parents {
		ref := parentRef{Short: shortSHA(p)}
		if inLog[p] {
			ref.Href = root + commitPage(p).href
		}
		view.Parents = append(view.Parents, ref)
	}
	for _, ch := range detail.Changes {
		view.Changes = append(view.Changes, changeRow{Path: ch.Path, Added: ch.Added, Deleted: ch.Deleted, Binary: ch.Binary})
	}
	body, err := renderBody(commitBodyTmpl, view)
	if err != nil {
		return err
	}
	return sc.renderPage(page.file, root, shortSHA(c.SHA)+" - source", body)
}

// renderDiff escapes a patch and marks added, removed, hunk, and header lines so
// the theme can color them. Invalid UTF-8 (from a truncation boundary or a
// binary-ish file) becomes the replacement character.
func renderDiff(patch []byte) template.HTML {
	if len(patch) == 0 {
		return ""
	}
	text := strings.ToValidUTF8(string(patch), "\uFFFD")
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		class := diffLineClass(line)
		esc := template.HTMLEscapeString(line)
		if class == "" {
			b.WriteString(esc)
		} else {
			fmt.Fprintf(&b, `<span class="%s">%s</span>`, class, esc)
		}
		b.WriteByte('\n')
	}
	return template.HTML(b.String()) //nolint:gosec // every line is HTML-escaped above
}

// diffLineClass picks the CSS class for one patch line.
func diffLineClass(line string) string {
	switch {
	case strings.HasPrefix(line, "diff --git "), strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "new file mode"), strings.HasPrefix(line, "deleted file mode"),
		strings.HasPrefix(line, "old mode"), strings.HasPrefix(line, "new mode"),
		strings.HasPrefix(line, "Binary files"), strings.HasPrefix(line, "+++ "),
		strings.HasPrefix(line, "--- "):
		return "hdr"
	case strings.HasPrefix(line, "@@"):
		return "hunk"
	case strings.HasPrefix(line, "+"):
		return "add"
	case strings.HasPrefix(line, "-"):
		return "del"
	}
	return ""
}

// ---- small helpers ---------------------------------------------------------

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func subjectOrPlaceholder(s string) string {
	if s == "" {
		return "(no subject)"
	}
	return s
}

// formatCommitDate renders a commit's author date in UTC, or "" if unknown.
func formatCommitDate(c gitsrc.Commit) string {
	if c.Date.IsZero() {
		return ""
	}
	return c.Date.Format("2006-01-02 15:04 UTC")
}

// humanSize formats a byte count for display.
func humanSize(n int64) string {
	switch {
	case n < 1<<10:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	}
}
