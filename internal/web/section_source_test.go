package web

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
	"github.com/major0/gitrack/internal/gitsrc"
)

// srcGit runs git in dir with extra environment, failing the test on error.
func srcGit(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newSourceRepo makes a tracker inside a real git repository with a fixed
// identity whose email must never appear in the generated site.
func newSourceRepo(t *testing.T) (*core.Repo, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	srcGit(t, dir, nil, "init", "-q", "-b", "trunk")
	srcGit(t, dir, nil, "config", "user.email", "test@example.com")
	srcGit(t, dir, nil, "config", "user.name", "Test User")
	return repo, dir
}

// commitAll stages everything and commits at a pinned date, returning the sha.
func commitAll(t *testing.T, dir, date, msg string) string {
	t.Helper()
	srcGit(t, dir, nil, "add", "-A")
	srcGit(t, dir, []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}, "commit", "-q", "-m", msg)
	return srcGit(t, dir, nil, "rev-parse", "HEAD")
}

func writeRepoText(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func generate(t *testing.T, repo *core.Repo) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "site")
	if _, err := Generate(repo, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return out
}

func TestSourceRendersTreeBlobLogAndCommit(t *testing.T) {
	repo, dir := newSourceRepo(t)
	id, err := repo.Create(core.NewTrackOpts{Title: "Parser fix"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	slug, _ := repo.GetField(id, "slug")

	writeRepoText(t, dir, "README.md", "# Hi\n")
	writeRepoText(t, dir, "src/main.go", "package main\n// <script>alert(1)</script>\n")
	writeRepoText(t, dir, "src/util/x.go", "package util\n")
	commitAll(t, dir, "2025-01-01T10:00:00Z", "first commit")
	writeRepoText(t, dir, "src/main.go", "package main\n// <script>alert(1)</script>\nvar x = 1\n")
	second := commitAll(t, dir, "2025-02-02T11:30:00Z", "Fix the parser\n\nCloses #"+slug)
	enableSections(t, repo, sectionIssues, sectionSource)
	// the sections file itself is part of the repo, so commit it before generating
	commitAll(t, dir, "2025-03-03T09:00:00Z", "enable the source section")
	out := generate(t, repo)

	// root tree: directories before files, relative links
	root := readOut(t, out, "source/index.html")
	di, fi := strings.Index(root, "src/"), strings.Index(root, "README.md")
	if di < 0 || fi < 0 || di > fi {
		t.Errorf("directories should be listed before files:\n%s", root)
	}
	if !strings.Contains(root, `href="../source/tree/src/index.html"`) {
		t.Errorf("root tree should link to the src directory page:\n%s", root)
	}
	if !strings.Contains(root, `href="../source/blob/README.md/index.html"`) {
		t.Errorf("root tree should link to the README blob page:\n%s", root)
	}

	// nested directory page: breadcrumbs climb back with the right depth
	srcDir := readOut(t, out, "source/tree/src/index.html")
	if !strings.Contains(srcDir, `href="../../../source/index.html"`) {
		t.Errorf("nested tree breadcrumb should reach the root with ../../../:\n%s", srcDir)
	}
	if !strings.Contains(srcDir, `href="../../../source/blob/src/main.go/index.html"`) {
		t.Errorf("nested tree should link its files:\n%s", srcDir)
	}
	if !strings.Contains(srcDir, `href="../../../source/tree/src/util/index.html"`) {
		t.Errorf("nested tree should link its subdirectory:\n%s", srcDir)
	}

	// blob: numbered lines, anchors, and HTML in the file is inert
	blob := readOut(t, out, "source/blob/src/main.go/index.html")
	if !strings.Contains(blob, `id="L1"`) || !strings.Contains(blob, `id="L3"`) {
		t.Errorf("blob should have per-line anchors:\n%s", blob)
	}
	if strings.Contains(blob, "<script>alert") {
		t.Errorf("file content must be escaped:\n%s", blob)
	}
	if !strings.Contains(blob, "&lt;script&gt;") {
		t.Errorf("escaped script tag expected:\n%s", blob)
	}

	// log: newest first, author name and date, link to commit pages
	log := readOut(t, out, "source/log.html")
	ni, oi := strings.Index(log, "enable the source section"), strings.Index(log, "first commit")
	if ni < 0 || oi < 0 || ni > oi {
		t.Errorf("log should list newest first:\n%s", log)
	}
	if !strings.Contains(log, "Test User") || !strings.Contains(log, "2025-02-02 11:30 UTC") {
		t.Errorf("log should show author name and date:\n%s", log)
	}
	if !strings.Contains(log, `href="../source/commit/`+second+`.html"`) {
		t.Errorf("log should link the commit page:\n%s", log)
	}

	// commit page: changed files, colored diff, and #ref autolinked to the track
	commit := readOut(t, out, "source/commit/"+second+".html")
	if !strings.Contains(commit, "src/main.go") || !strings.Contains(commit, `class="add"`) {
		t.Errorf("commit page should list the file and show added lines:\n%s", commit)
	}
	if !strings.Contains(commit, `href="../../track/`+id+`.html"`) {
		t.Errorf("#ref in the commit message should link to its track:\n%s", commit)
	}

	// top nav
	home := readOut(t, out, "index.html")
	if !strings.Contains(home, `href="source/index.html">source`) {
		t.Errorf("nav should have a source entry:\n%s", home)
	}

	// privacy: the author's email address appears nowhere in the site
	err = filepath.WalkDir(out, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if strings.Contains(string(b), "@example.com") {
			t.Errorf("an email address leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func TestSourceBlobKinds(t *testing.T) {
	repo, dir := newSourceRepo(t)
	writeRepoText(t, dir, "bin.dat", "abc\x00def")
	writeRepoText(t, dir, "empty.txt", "")
	writeRepoText(t, dir, "big.txt", strings.Repeat(strings.Repeat("x", 99)+"\n", 6000)) // ~600 KB
	writeRepoText(t, dir, "dir with space/a#b %c.txt", "odd name\n")
	writeRepoText(t, dir, "a", "a file named a\n")
	writeRepoText(t, dir, "a.html/b", "inside a directory named a.html\n")
	if err := os.Symlink("empty.txt", filepath.Join(dir, "link")); err != nil {
		t.Logf("symlinks unavailable, skipping that case: %v", err)
	}
	sha := commitAll(t, dir, "2025-01-01T00:00:00Z", "kinds")
	enableSections(t, repo, sectionSource)
	out := generate(t, repo)

	if page := readOut(t, out, "source/blob/bin.dat/index.html"); !strings.Contains(page, "binary file") {
		t.Errorf("binary file should say so:\n%s", page)
	}
	if page := readOut(t, out, "source/blob/empty.txt/index.html"); !strings.Contains(page, "empty file") {
		t.Errorf("empty file should say so:\n%s", page)
	}
	big := readOut(t, out, "source/blob/big.txt/index.html")
	if !strings.Contains(big, "showing the first 512.0 KiB") {
		t.Errorf("oversized file should be truncated with a note:\n%.400s", big)
	}
	if strings.Contains(big, `id="L6000"`) {
		t.Errorf("the truncated page should not contain the last line")
	}

	// odd file names: the page exists at the raw path and is linked escaped
	if _, err := os.Stat(filepath.Join(out, "source", "blob", "dir with space", "a#b %c.txt", "index.html")); err != nil {
		t.Errorf("page for an odd file name missing: %v", err)
	}
	tree := readOut(t, out, "source/tree/dir with space/index.html")
	if !strings.Contains(tree, "dir%20with%20space/a%23b%20%25c.txt/index.html") {
		t.Errorf("odd file name should be linked URL-escaped:\n%s", tree)
	}

	// a file `a` and a directory `a.html/` must not collide
	for _, rel := range []string{"source/blob/a/index.html", "source/blob/a.html/b/index.html"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s missing: %v", rel, err)
		}
	}

	if _, err := os.Lstat(filepath.Join(dir, "link")); err == nil {
		page := readOut(t, out, "source/blob/link/index.html")
		if !strings.Contains(page, "symbolic link to empty.txt") {
			t.Errorf("symlink page should show its target:\n%s", page)
		}
	}

	// the huge file makes the commit's patch exceed its limit
	commit := readOut(t, out, "source/commit/"+sha+".html")
	if !strings.Contains(commit, "cut short") {
		t.Errorf("an oversized patch should carry a truncation note")
	}
}

func TestSourceRefAndCommitsConfig(t *testing.T) {
	repo, dir := newSourceRepo(t)
	writeRepoText(t, dir, "old.txt", "old\n")
	commitAll(t, dir, "2025-01-01T00:00:00Z", "one")
	srcGit(t, dir, nil, "tag", "v1")
	writeRepoText(t, dir, "new.txt", "new\n")
	commitAll(t, dir, "2025-01-02T00:00:00Z", "two")
	writeRepoText(t, dir, "newer.txt", "newer\n")
	commitAll(t, dir, "2025-01-03T00:00:00Z", "three")

	enableSections(t, repo, sectionSource)
	writeSiteFile(t, repo, "source/ref", "v1\n")
	writeSiteFile(t, repo, "source/commits", "1\n")
	out := generate(t, repo)

	root := readOut(t, out, "source/index.html")
	if !strings.Contains(root, "old.txt") || strings.Contains(root, "new.txt") {
		t.Errorf("the v1 snapshot should contain only old.txt:\n%s", root)
	}
	entries, err := os.ReadDir(filepath.Join(out, "source", "commit"))
	if err != nil {
		t.Fatalf("read commit dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("commits=1 should render one commit page, got %d", len(entries))
	}

	// a malformed commits value falls back to the default instead of failing
	writeSiteFile(t, repo, "source/commits", "lots\n")
	writeSiteFile(t, repo, "source/ref", "HEAD\n")
	out2 := generate(t, repo)
	entries, err = os.ReadDir(filepath.Join(out2, "source", "commit"))
	if err != nil {
		t.Fatalf("read commit dir: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("default limit should render all 3 commits, got %d", len(entries))
	}
}

func TestSourceEmitsNothingWithoutAnythingToBrowse(t *testing.T) {
	// 1. a tracker that is not inside a git repository
	notGit := t.TempDir()
	repo, err := core.Init(notGit)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	enableSections(t, repo, sectionIssues, sectionSource)
	out := generate(t, repo)
	assertNoSource(t, out, "a non-git directory")

	// 2. a git repository with no commits yet
	empty, _ := newSourceRepo(t)
	enableSections(t, empty, sectionIssues, sectionSource)
	assertNoSource(t, generate(t, empty), "an empty history")

	// 3. a ref that does not resolve, and one that looks like a git option
	repo3, dir3 := newSourceRepo(t)
	writeRepoText(t, dir3, "f", "x\n")
	commitAll(t, dir3, "2025-01-01T00:00:00Z", "one")
	enableSections(t, repo3, sectionIssues, sectionSource)
	for _, bad := range []string{"no-such-ref", "--output=x"} {
		writeSiteFile(t, repo3, "source/ref", bad+"\n")
		assertNoSource(t, generate(t, repo3), "ref "+bad)
	}
}

func assertNoSource(t *testing.T, out, why string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(out, "source")); !os.IsNotExist(err) {
		t.Errorf("%s: no source/ output expected (err=%v)", why, err)
	}
	home, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("%s: read index: %v", why, err)
	}
	if strings.Contains(string(home), "source/index.html") {
		t.Errorf("%s: no source nav entry expected:\n%s", why, home)
	}
}

func TestIndexTreeSkipsUnsafePaths(t *testing.T) {
	entries := []gitsrc.Entry{
		{Path: "ok/file.txt", Type: "blob", Mode: "100644"},
		{Path: "../escape.txt", Type: "blob", Mode: "100644"},
		{Path: "a/../b.txt", Type: "blob", Mode: "100644"},
		{Path: "a//b.txt", Type: "blob", Mode: "100644"},
		{Path: "./dot.txt", Type: "blob", Mode: "100644"},
	}
	dirs, valid := indexTree(entries)
	if len(valid) != 1 || valid[0].Path != "ok/file.txt" {
		t.Errorf("only the safe path should be accepted, got %+v", valid)
	}
	if !dirs[""].subdirs["ok"] || len(dirs) != 2 {
		t.Errorf("unexpected directory index: %+v", dirs)
	}
}

func TestSourceHelpers(t *testing.T) {
	if got := escapePath("dir with space/a#b%c"); got != "dir%20with%20space/a%23b%25c" {
		t.Errorf("escapePath = %q", got)
	}
	if got := string(renderCode([]byte("a\nb\n"), nil, "").HTML); strings.Count(got, `class="src-line"`) != 2 {
		t.Errorf("a trailing newline must not add a line: %q", got)
	}
	if got := string(renderCode([]byte("<b>"), nil, "").HTML); !strings.Contains(got, "&lt;b&gt;") {
		t.Errorf("renderCode must escape: %q", got)
	}
	cases := map[string]string{
		"diff --git a/x b/x": "hdr", "@@ -1 +1 @@": "hunk", "+added": "add",
		"-removed": "del", " context": "", "+++ b/x": "hdr", "--- a/x": "hdr",
	}
	for line, want := range cases {
		if got := diffLineClass(line); got != want {
			t.Errorf("diffLineClass(%q) = %q, want %q", line, got, want)
		}
	}
	if got := humanSize(2048); got != "2.0 KiB" {
		t.Errorf("humanSize(2048) = %q", got)
	}
}

// assertMentionLink checks how a track page shows one mentioning commit: linked
// to a commit page that really exists in the output, or plain with no link.
func assertMentionLink(t *testing.T, page, out string, m core.Mention, linked bool) {
	t.Helper()
	link := `<a href="../source/commit/` + m.SHA + `.html"><code class="sha">` + m.Short + `</code></a>`
	if linked {
		if !strings.Contains(page, link) {
			t.Errorf("mention %s should link to its commit page:\n%s", m.Short, page)
		}
		if _, err := os.Stat(filepath.Join(out, "source", "commit", m.SHA+".html")); err != nil {
			t.Errorf("mention %s links to a commit page that does not exist: %v", m.Short, err)
		}
		return
	}
	// the unlinked form is the same entry without the anchor
	if !strings.Contains(page, `<li><code class="sha">`+m.Short+`</code> <span class="verb">`) {
		t.Errorf("mention %s should still be listed in the original plain form:\n%s", m.Short, page)
	}
	if strings.Contains(page, "source/commit/"+m.SHA) {
		t.Errorf("mention %s has no commit page and must not link:\n%s", m.Short, page)
	}
}

func TestTrackMentionsLinkToSourceCommitPages(t *testing.T) {
	repo, dir := newSourceRepo(t)
	id, err := repo.Create(core.NewTrackOpts{Title: "Parser fix"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	slug, _ := repo.GetField(id, "slug")

	writeRepoText(t, dir, "a.txt", "1\n")
	early := commitAll(t, dir, "2025-01-01T00:00:00Z", "start work on #"+slug)
	srcGit(t, dir, nil, "tag", "v1")
	writeRepoText(t, dir, "a.txt", "2\n")
	commitAll(t, dir, "2025-01-02T00:00:00Z", "an unrelated change")
	writeRepoText(t, dir, "a.txt", "3\n")
	late := commitAll(t, dir, "2025-01-03T00:00:00Z", "Fix it\n\ncloses #"+slug)

	mentions, err := repo.Mentions(id)
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	by := map[string]core.Mention{}
	for _, m := range mentions {
		by[m.SHA] = m
	}
	if len(by) != 2 || by[early].SHA == "" || by[late].SHA == "" {
		t.Fatalf("expected the early and late commits to mention the track, got %+v", mentions)
	}
	trackPage := func(out string) string { return readOut(t, out, "track/"+id+".html") }

	// 1. issues + source, default log: both commits are in the log, so both link
	enableSections(t, repo, sectionIssues, sectionSource)
	out := generate(t, repo)
	assertMentionLink(t, trackPage(out), out, by[early], true)
	assertMentionLink(t, trackPage(out), out, by[late], true)

	// 2. a log of one commit holds only HEAD: the older mention is plain text
	writeSiteFile(t, repo, "source/commits", "1\n")
	out = generate(t, repo)
	assertMentionLink(t, trackPage(out), out, by[late], true)
	assertMentionLink(t, trackPage(out), out, by[early], false)

	// 3. a snapshot of the older tag: the later commit is not in its history
	writeSiteFile(t, repo, "source/commits", "100\n")
	writeSiteFile(t, repo, "source/ref", "v1\n")
	out = generate(t, repo)
	assertMentionLink(t, trackPage(out), out, by[early], true)
	assertMentionLink(t, trackPage(out), out, by[late], false)

	// 4. without the source section there are no commit pages, so nothing links
	enableSections(t, repo, sectionIssues)
	out = generate(t, repo)
	assertMentionLink(t, trackPage(out), out, by[early], false)
	assertMentionLink(t, trackPage(out), out, by[late], false)
}

// readOutMaybe is readOut that reports a missing file instead of failing the test.
func readOutMaybe(out, rel string) (string, error) {
	b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
	return string(b), err
}
