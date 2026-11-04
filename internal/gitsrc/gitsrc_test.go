package gitsrc

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// git runs git in dir with extra environment, failing the test on error, and
// returns its trimmed stdout.
func git(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo makes an empty git repository with a fixed identity.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, nil, "init", "-q", "-b", "trunk")
	git(t, dir, nil, "config", "user.email", "ada@example.com")
	git(t, dir, nil, "config", "user.name", "Ada Lovelace")
	return dir
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// commit stages everything and commits with a pinned date, returning the sha.
func commit(t *testing.T, dir, date, msg string) string {
	t.Helper()
	git(t, dir, nil, "add", "-A")
	env := []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	git(t, dir, env, "commit", "-q", "-m", msg)
	return git(t, dir, nil, "rev-parse", "HEAD")
}

func TestResolveCommit(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "hi\n")
	want := commit(t, dir, "2025-01-01T00:00:00Z", "first")
	r := Open(dir)

	got, err := r.ResolveCommit("")
	if err != nil || got != want {
		t.Errorf("ResolveCommit(\"\") = %q, %v; want %q", got, err, want)
	}
	git(t, dir, nil, "tag", "v1")
	if got, err := r.ResolveCommit("v1"); err != nil || got != want {
		t.Errorf("ResolveCommit(v1) = %q, %v; want %q", got, err, want)
	}
	for _, bad := range []string{"no-such-ref", "-n1", "--output=x"} {
		if _, err := r.ResolveCommit(bad); !errors.Is(err, ErrNoCommit) {
			t.Errorf("ResolveCommit(%q) error = %v, want ErrNoCommit", bad, err)
		}
	}
}

func TestResolveCommitNotARepoOrEmpty(t *testing.T) {
	if _, err := Open(t.TempDir()).ResolveCommit("HEAD"); !errors.Is(err, ErrNoCommit) {
		t.Errorf("non-repo error = %v, want ErrNoCommit", err)
	}
	empty := newRepo(t) // initialized, no commits
	if _, err := Open(empty).ResolveCommit("HEAD"); !errors.Is(err, ErrNoCommit) {
		t.Errorf("empty-history error = %v, want ErrNoCommit", err)
	}
}

func TestFilesListsBlobsSymlinksAndSubmodules(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "README.md", "hello\n")
	write(t, dir, "src/deep/main.go", "package main\n")
	if err := os.Symlink("README.md", filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// Stage the files first, THEN register a submodule entry (a gitlink) without
	// needing a real submodule. Re-running `add -A` afterwards would stage its
	// removal, since no vendor/sub exists on disk.
	git(t, dir, nil, "add", "-A")
	git(t, dir, nil, "update-index", "--add", "--cacheinfo",
		"160000,4b825dc642cb6eb9a060e54bf8d69288fbee4904,vendor/sub")
	date := "2025-01-01T00:00:00Z"
	git(t, dir, []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date},
		"commit", "-q", "-m", "first")
	sha := git(t, dir, nil, "rev-parse", "HEAD")

	files, err := Open(dir).Files(sha)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	by := map[string]Entry{}
	for _, e := range files {
		by[e.Path] = e
	}
	if e := by["README.md"]; e.Size != 6 || e.Type != "blob" {
		t.Errorf("README.md = %+v, want a 6-byte blob", e)
	}
	if _, ok := by["src/deep/main.go"]; !ok {
		t.Errorf("nested file missing from %v", files)
	}
	if !by["link"].IsSymlink() {
		t.Errorf("link should be a symlink: %+v", by["link"])
	}
	if sub := by["vendor/sub"]; !sub.IsSubmodule() || sub.Size != 0 {
		t.Errorf("vendor/sub = %+v, want a submodule of size 0", sub)
	}
}

func TestBlobReaderKeepsStreamInStepAfterTruncation(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "big.txt", strings.Repeat("0123456789\n", 100)) // 1100 bytes
	write(t, dir, "small.txt", "tiny\n")
	write(t, dir, "bin.dat", "abc\x00def")
	sha := commit(t, dir, "2025-01-01T00:00:00Z", "first")

	files, err := Open(dir).Files(sha)
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	by := map[string]Entry{}
	for _, e := range files {
		by[e.Path] = e
	}

	br, err := Open(dir).OpenBlobs()
	if err != nil {
		t.Fatalf("OpenBlobs: %v", err)
	}
	defer func() { _ = br.Close() }()

	big, err := br.Read(by["big.txt"].SHA, 50)
	if err != nil {
		t.Fatalf("Read big: %v", err)
	}
	if !big.Truncated || len(big.Data) != 50 || big.Size != 1100 {
		t.Errorf("big = trunc %v, %d bytes, size %d; want truncated, 50, 1100", big.Truncated, len(big.Data), big.Size)
	}
	// the read after a truncated one must still be correct (stream in step)
	small, err := br.Read(by["small.txt"].SHA, 1000)
	if err != nil {
		t.Fatalf("Read small: %v", err)
	}
	if string(small.Data) != "tiny\n" || small.Truncated || small.Binary {
		t.Errorf("small = %+v, want intact text", small)
	}
	bin, err := br.Read(by["bin.dat"].SHA, 1000)
	if err != nil {
		t.Fatalf("Read bin: %v", err)
	}
	if !bin.Binary {
		t.Errorf("a NUL byte should mark the blob binary")
	}
	if _, err := br.Read("0000000000000000000000000000000000000000", 10); err == nil {
		t.Errorf("reading a missing object should fail")
	}
}

func TestLogNewestFirstWithFieldsAndNoEmail(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a", "1")
	commit(t, dir, "2025-01-01T10:00:00Z", "first")
	write(t, dir, "a", "2")
	commit(t, dir, "2025-02-02T11:30:00Z", "second subject\n\nBody line one.\nBody line two.\n\ncloses #x")
	write(t, dir, "a", "3")
	head := commit(t, dir, "2025-03-03T12:00:00Z", "third")

	r := Open(dir)
	all, err := r.Log(head, 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(all) != 3 || all[0].Subject != "third" || all[2].Subject != "first" {
		t.Fatalf("log order/len wrong: %+v", all)
	}
	second := all[1]
	if second.Subject != "second subject" {
		t.Errorf("Subject = %q", second.Subject)
	}
	if want := "Body line one.\nBody line two.\n\ncloses #x"; second.Body != want {
		t.Errorf("Body = %q, want %q", second.Body, want)
	}
	if second.Author != "Ada Lovelace" {
		t.Errorf("Author = %q", second.Author)
	}
	if got := second.Date.Format("2006-01-02 15:04"); got != "2025-02-02 11:30" {
		t.Errorf("Date = %q", got)
	}
	if len(second.Parents) != 1 || second.Parents[0] != all[2].SHA {
		t.Errorf("Parents = %v, want [%s]", second.Parents, all[2].SHA)
	}
	if len(all[2].Parents) != 0 {
		t.Errorf("root commit has parents: %v", all[2].Parents)
	}
	for _, c := range all {
		if strings.Contains(c.Author+c.Subject+c.Body, "@") && !strings.Contains(c.Body, "#") {
			t.Errorf("an email address leaked into %+v", c)
		}
	}
	if limited, _ := r.Log(head, 2); len(limited) != 2 {
		t.Errorf("Log(…, 2) returned %d commits", len(limited))
	}
	if none, _ := r.Log(head, 0); len(none) != 0 {
		t.Errorf("Log(…, 0) should be empty")
	}
}

func TestDescribeRootCommitAndModification(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a.txt", "one\ntwo\n")
	write(t, dir, "img.bin", "x\x00y")
	rootSHA := commit(t, dir, "2025-01-01T00:00:00Z", "root")
	write(t, dir, "a.txt", "one\nTWO\nthree\n")
	modSHA := commit(t, dir, "2025-01-02T00:00:00Z", "modify")

	r := Open(dir)
	commits, err := r.Log(modSHA, 5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	byS := map[string]Commit{}
	for _, c := range commits {
		byS[c.SHA] = c
	}

	root, err := r.Describe(byS[rootSHA], 1<<20)
	if err != nil {
		t.Fatalf("Describe root: %v", err)
	}
	if len(root.Changes) != 2 {
		t.Fatalf("root changes = %+v, want 2 files", root.Changes)
	}
	changes := map[string]FileChange{}
	for _, c := range root.Changes {
		changes[c.Path] = c
	}
	if c := changes["a.txt"]; c.Added != 2 || c.Deleted != 0 || c.Binary {
		t.Errorf("a.txt = %+v, want +2 -0", c)
	}
	if !changes["img.bin"].Binary {
		t.Errorf("img.bin should be binary: %+v", changes["img.bin"])
	}
	if !strings.Contains(string(root.Patch), "+one") {
		t.Errorf("root patch missing added line:\n%s", root.Patch)
	}

	mod, err := r.Describe(byS[modSHA], 1<<20)
	if err != nil {
		t.Fatalf("Describe modify: %v", err)
	}
	if len(mod.Changes) != 1 || mod.Changes[0].Added != 2 || mod.Changes[0].Deleted != 1 {
		t.Errorf("modify changes = %+v, want a.txt +2 -1", mod.Changes)
	}
	if !strings.Contains(string(mod.Patch), "-two") || !strings.Contains(string(mod.Patch), "+TWO") {
		t.Errorf("modify patch missing the change:\n%s", mod.Patch)
	}
}

func TestDescribeTruncatesPatch(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "big.txt", strings.Repeat("a line of text\n", 5000))
	sha := commit(t, dir, "2025-01-01T00:00:00Z", "big")
	r := Open(dir)
	commits, _ := r.Log(sha, 1)

	d, err := r.Describe(commits[0], 1000)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if !d.PatchTruncated || len(d.Patch) != 1000 {
		t.Errorf("patch = truncated %v, %d bytes; want truncated, 1000", d.PatchTruncated, len(d.Patch))
	}
	// the numstat summary is independent of the patch limit
	if len(d.Changes) != 1 || d.Changes[0].Added != 5000 {
		t.Errorf("changes = %+v, want big.txt +5000", d.Changes)
	}
}

func TestDescribeMergeUsesFirstParent(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "base.txt", "base\n")
	commit(t, dir, "2025-01-01T00:00:00Z", "base")
	git(t, dir, nil, "checkout", "-q", "-b", "topic")
	write(t, dir, "topic.txt", "from topic\n")
	commit(t, dir, "2025-01-02T00:00:00Z", "topic work")
	git(t, dir, nil, "checkout", "-q", "trunk")
	write(t, dir, "trunk.txt", "from trunk\n")
	commit(t, dir, "2025-01-03T00:00:00Z", "trunk work")
	env := []string{"GIT_AUTHOR_DATE=2025-01-04T00:00:00Z", "GIT_COMMITTER_DATE=2025-01-04T00:00:00Z"}
	git(t, dir, env, "merge", "-q", "--no-ff", "-m", "merge topic", "topic")
	merge := git(t, dir, nil, "rev-parse", "HEAD")

	r := Open(dir)
	commits, err := r.Log(merge, 1)
	if err != nil || len(commits) != 1 {
		t.Fatalf("Log: %v %v", commits, err)
	}
	if len(commits[0].Parents) != 2 {
		t.Fatalf("expected a merge commit, parents = %v", commits[0].Parents)
	}
	d, err := r.Describe(commits[0], 1<<20)
	if err != nil {
		t.Fatalf("Describe merge: %v", err)
	}
	// against the first parent (trunk), the merge brought in only topic.txt
	if len(d.Changes) != 1 || d.Changes[0].Path != "topic.txt" {
		t.Errorf("merge changes = %+v, want only topic.txt", d.Changes)
	}
}
