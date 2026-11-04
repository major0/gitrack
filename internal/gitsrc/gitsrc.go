// Package gitsrc reads a git repository for the static source browser
// (ADR-0042): resolve a ref, list a tree, read blobs, walk the log, and describe
// one commit. It shells out to git and knows nothing about the tracker.
//
// Every call that can run a configured driver passes --no-ext-diff and
// --no-textconv, and no ref is ever allowed to start with "-", so a browser run
// never executes repository-supplied commands or reads data as an option.
package gitsrc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ErrNoCommit is returned when a ref does not resolve to a commit: the path is
// not a git repository, the history is empty, the ref does not exist, or the ref
// was rejected (it must not start with "-").
var ErrNoCommit = errors.New("no such commit")

// binarySniff is how many leading bytes are checked for a NUL, git's own
// heuristic for "this is a binary file".
const binarySniff = 8000

// Repo is a handle to a git working directory.
type Repo struct {
	root string
}

// Open returns a handle for the git repository containing root. It does not
// touch git; problems surface from the first call that needs it.
func Open(root string) *Repo { return &Repo{root: root} }

// git builds a command that runs in the repository.
func (r *Repo) git(args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"-C", r.root}, args...)...)
}

// output runs git and returns stdout, folding stderr into any error.
func (r *Repo) output(args ...string) ([]byte, error) {
	cmd := r.git(args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// ResolveCommit resolves ref (default "HEAD") to a full commit sha. A ref that
// starts with "-" is rejected so it can never be parsed as a git option.
func (r *Repo) ResolveCommit(ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	if strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("%w: ref %q must not start with '-'", ErrNoCommit, ref)
	}
	out, err := r.output("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNoCommit, ref)
	}
	return strings.TrimSpace(string(out)), nil
}

// Entry is one non-directory item in a tree: a file, a symlink, or a submodule.
type Entry struct {
	Mode string // e.g. "100644"
	Type string // "blob" or "commit" (a submodule)
	SHA  string
	Size int64 // 0 for a submodule
	Path string
}

// IsSymlink reports whether the entry is a symbolic link.
func (e Entry) IsSymlink() bool { return e.Mode == "120000" }

// IsSubmodule reports whether the entry is a submodule (a gitlink).
func (e Entry) IsSubmodule() bool { return e.Type == "commit" }

// Files lists every non-directory entry in commit's tree, recursively. Paths are
// relative to the repository top level.
func (r *Repo) Files(commit string) ([]Entry, error) {
	out, err := r.output("ls-tree", "-r", "-z", "--long", "--full-tree", commit)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		tab := strings.IndexByte(rec, '\t')
		if tab < 0 {
			return nil, fmt.Errorf("ls-tree: malformed record %q", rec)
		}
		meta := strings.Fields(rec[:tab])
		if len(meta) != 4 {
			return nil, fmt.Errorf("ls-tree: malformed record %q", rec)
		}
		e := Entry{Mode: meta[0], Type: meta[1], SHA: meta[2], Path: rec[tab+1:]}
		if meta[3] != "-" {
			n, err := strconv.ParseInt(meta[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("ls-tree: bad size %q: %w", meta[3], err)
			}
			e.Size = n
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Blob is the content of a file, possibly cut short.
type Blob struct {
	Data      []byte
	Size      int64 // full size in the repository
	Truncated bool  // Data holds only the first part of the file
	Binary    bool  // a NUL appears in the first binarySniff bytes
}

// BlobReader reads many blobs through one long-lived `git cat-file --batch`
// process, rather than starting a process per file.
type BlobReader struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

// OpenBlobs starts a blob reader. The caller must Close it.
func (r *Repo) OpenBlobs() (*BlobReader, error) {
	cmd := r.git("cat-file", "--batch")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting git cat-file: %w", err)
	}
	return &BlobReader{cmd: cmd, in: in, out: bufio.NewReader(out)}, nil
}

// Read returns the blob with the given object id, keeping at most limit bytes.
// The rest of an oversized blob is consumed and discarded so the stream stays in
// step for the next request.
func (b *BlobReader) Read(sha string, limit int) (Blob, error) {
	if _, err := io.WriteString(b.in, sha+"\n"); err != nil {
		return Blob{}, fmt.Errorf("requesting %s: %w", sha, err)
	}
	header, err := b.out.ReadString('\n')
	if err != nil {
		return Blob{}, fmt.Errorf("reading header for %s: %w", sha, err)
	}
	fields := strings.Fields(header)
	if len(fields) == 2 && fields[1] == "missing" {
		return Blob{}, fmt.Errorf("object %s not found", sha)
	}
	if len(fields) != 3 {
		return Blob{}, fmt.Errorf("unexpected cat-file header %q", strings.TrimSpace(header))
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return Blob{}, fmt.Errorf("bad size in header %q: %w", strings.TrimSpace(header), err)
	}
	keep := size
	if keep > int64(limit) {
		keep = int64(limit)
	}
	data := make([]byte, keep)
	if _, err := io.ReadFull(b.out, data); err != nil {
		return Blob{}, fmt.Errorf("reading %s: %w", sha, err)
	}
	// Discard the unread remainder plus the newline that follows every object.
	if _, err := io.CopyN(io.Discard, b.out, size-keep+1); err != nil {
		return Blob{}, fmt.Errorf("skipping rest of %s: %w", sha, err)
	}
	sniff := data
	if len(sniff) > binarySniff {
		sniff = sniff[:binarySniff]
	}
	return Blob{
		Data:      data,
		Size:      size,
		Truncated: size > keep,
		Binary:    bytes.IndexByte(sniff, 0) >= 0,
	}, nil
}

// Close shuts the blob reader down.
func (b *BlobReader) Close() error {
	_ = b.in.Close()
	return b.cmd.Wait()
}

// Commit is one entry of the log.
type Commit struct {
	SHA     string
	Parents []string
	Author  string // name only; email addresses are never read (ADR-0042)
	Date    time.Time
	Subject string
	Body    string
}

// Field and record separators for the log format; neither can occur in a name,
// a date, or sha, and both are vanishingly rare in a commit message.
const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// Log returns up to n commits reachable from commit, newest first.
func (r *Repo) Log(commit string, n int) ([]Commit, error) {
	if n <= 0 {
		return nil, nil
	}
	// sha, parents, author name, author date (ISO), full message; fields end in
	// 0x1f and each record in 0x1e. The author email (%ae) is deliberately absent.
	const format = "--format=%H%x1f%P%x1f%an%x1f%aI%x1f%B%x1e"
	out, err := r.output("log", "-n", strconv.Itoa(n), "--no-color", format, commit, "--")
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, rec := range strings.Split(string(out), recordSep) {
		rec = strings.TrimLeft(rec, "\n")
		if strings.TrimSpace(rec) == "" {
			continue
		}
		f := strings.SplitN(rec, fieldSep, 5)
		if len(f) != 5 {
			return nil, fmt.Errorf("log: malformed record %q", rec)
		}
		c := Commit{SHA: f[0], Parents: strings.Fields(f[1]), Author: f[2]}
		if t, err := time.Parse(time.RFC3339, f[3]); err == nil {
			c.Date = t.UTC()
		}
		subject, body, _ := strings.Cut(strings.TrimSpace(f[4]), "\n")
		c.Subject = subject
		c.Body = strings.TrimSpace(body)
		commits = append(commits, c)
	}
	return commits, nil
}

// FileChange is one file's line counts in a commit.
type FileChange struct {
	Path    string
	Added   int
	Deleted int
	Binary  bool
}

// Detail is what one commit changed.
type Detail struct {
	Changes        []FileChange
	Patch          []byte
	PatchTruncated bool
}

// emptyTree asks git for the empty tree's id, which differs between sha1 and
// sha256 repositories.
func (r *Repo) emptyTree() (string, error) {
	cmd := r.git("hash-object", "-t", "tree", "--stdin")
	cmd.Stdin = strings.NewReader("")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git hash-object: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// Describe summarizes what c changed against its first parent (a merge is shown
// against its first parent; a root commit against the empty tree). The patch is
// cut at patchLimit bytes.
func (r *Repo) Describe(c Commit, patchLimit int) (Detail, error) {
	base := ""
	if len(c.Parents) > 0 {
		base = c.Parents[0]
	} else {
		var err error
		if base, err = r.emptyTree(); err != nil {
			return Detail{}, err
		}
	}
	common := []string{"--no-ext-diff", "--no-textconv", "--no-renames"}

	stat, err := r.output(append(append([]string{"diff", "--numstat", "-z"}, common...), base, c.SHA, "--")...)
	if err != nil {
		return Detail{}, err
	}
	d := Detail{}
	for _, rec := range strings.Split(string(stat), "\x00") {
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\t", 3)
		if len(parts) != 3 {
			return Detail{}, fmt.Errorf("diff --numstat: malformed record %q", rec)
		}
		fc := FileChange{Path: parts[2]}
		if parts[0] == "-" {
			fc.Binary = true
		} else {
			fc.Added, _ = strconv.Atoi(parts[0])
			fc.Deleted, _ = strconv.Atoi(parts[1])
		}
		d.Changes = append(d.Changes, fc)
	}

	patchArgs := append(append([]string{"diff", "--no-color"}, common...), base, c.SHA, "--")
	d.Patch, d.PatchTruncated, err = r.outputLimited(patchLimit, patchArgs...)
	if err != nil {
		return Detail{}, err
	}
	return d, nil
}

// outputLimited reads at most limit bytes of git's stdout. If the output is
// longer, the git process is stopped rather than left to produce the rest.
func (r *Repo) outputLimited(limit int, args ...string) ([]byte, bool, error) {
	cmd := r.git(args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, false, fmt.Errorf("starting git %s: %w", args[0], err)
	}
	buf, readErr := io.ReadAll(io.LimitReader(stdout, int64(limit)+1))
	truncated := len(buf) > limit
	if truncated {
		buf = buf[:limit]
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	switch {
	case truncated:
		return buf, true, nil
	case readErr != nil:
		return nil, false, readErr
	case waitErr != nil:
		return nil, false, fmt.Errorf("git %s: %w: %s", args[0], waitErr, strings.TrimSpace(stderr.String()))
	}
	return buf, false, nil
}
