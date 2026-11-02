package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CommentSuffix is the storage suffix for comment files: a comment is a Markdown
// document (ADR-0023). The suffix is never part of the addressable id (ADR-0022).
const CommentSuffix = ".md"

// Comment is a materialized comment: its id, optional reply-to parent, and body.
// Author and date are NOT stored; they are derived from git (ADR-0023).
type Comment struct {
	ID      string
	ReplyTo string
	Body    string
}

// commentsDir is the absolute comments directory for a track.
func (r *Repo) commentsDir(id string) string {
	return filepath.Join(r.trackPath(id), "comments")
}

// AddComment writes a new comment on track id with the given Markdown body and
// optional reply-to parent, minting a UUIDv7 name (ADR-0023). It returns the new
// comment's id. The body is stored verbatim; any #refs/URLs in it are rendered
// by the web/issues generator (ADR-0011), not at write time.
func (r *Repo) AddComment(id, body, replyTo string) (string, error) {
	if _, err := os.Stat(r.trackPath(id)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrNoTrack, id)
	}
	if replyTo != "" {
		if _, err := os.Stat(r.commentPath(id, replyTo)); err != nil {
			return "", fmt.Errorf("reply-to comment %q not found on this track", replyTo)
		}
	}
	cid, err := NewUUIDv7()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(r.commentsDir(id), 0o755); err != nil {
		return "", fmt.Errorf("creating comments dir: %w", err)
	}
	content := renderComment(replyTo, body)
	if err := os.WriteFile(r.commentPath(id, cid), []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing comment: %w", err)
	}
	return cid, nil
}

// commentPath returns the file path for a comment id on a track.
func (r *Repo) commentPath(id, cid string) string {
	return filepath.Join(r.commentsDir(id), cid+CommentSuffix)
}

// LoadComment reads and parses a comment file (front matter + body, ADR-0023).
func (r *Repo) LoadComment(id, cid string) (*Comment, error) {
	b, err := os.ReadFile(r.commentPath(id, cid))
	if err != nil {
		return nil, fmt.Errorf("reading comment %q: %w", cid, err)
	}
	replyTo, body := parseComment(string(b))
	return &Comment{ID: cid, ReplyTo: replyTo, Body: body}, nil
}

// renderComment serializes a comment: a reply carries a minimal front-matter
// block with only reply-to; a top-level comment is pure body (ADR-0023).
func renderComment(replyTo, body string) string {
	body = strings.TrimRight(body, "\n")
	if replyTo == "" {
		return body + "\n"
	}
	return fmt.Sprintf("---\nreply-to: %s\n---\n%s\n", replyTo, body)
}

// parseComment splits an optional YAML front-matter block (only reply-to is
// recognized) from the Markdown body. Tolerates "front matter + body" and
// "body only" (ADR-0023).
func parseComment(content string) (replyTo, body string) {
	if !strings.HasPrefix(content, "---\n") {
		return "", strings.TrimRight(content, "\n")
	}
	rest := content[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		// malformed front matter; treat the whole thing as body
		return "", strings.TrimRight(content, "\n")
	}
	fm := rest[:end]
	body = rest[end+len("\n---\n"):]
	for _, line := range strings.Split(fm, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "reply-to:"); ok {
			replyTo = strings.TrimSpace(v)
		}
	}
	return replyTo, strings.TrimRight(body, "\n")
}

// CiteQuote renders a comment's body as a Markdown blockquote, for seeding the
// editor when replying (git-commit style). Each body line is prefixed with "> ".
func CiteQuote(body string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		if line == "" {
			b.WriteString(">\n")
		} else {
			b.WriteString("> ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// Author and Created are derived from git history of the comment file (ADR-0023);
// they are not stored. CommentMeta carries them alongside the loaded comment.
type CommentMeta struct {
	Comment
	Author  string // "Name <email>" of the commit that added the file, or ""
	Created string // RFC3339 date the file was added, or ""
}

// ListComments loads every comment on a track with its git-derived author and
// created date (ADR-0023). Author/date come from `git log` on the comment file;
// if history is unavailable (e.g. not yet committed), they are empty and the
// comment still renders.
func (r *Repo) ListComments(id string) ([]CommentMeta, error) {
	names, err := r.listMulti(id, "comments")
	if err != nil {
		return nil, err
	}
	out := make([]CommentMeta, 0, len(names))
	for _, name := range names {
		cid := strings.TrimSuffix(name, CommentSuffix)
		c, err := r.LoadComment(id, cid)
		if err != nil {
			return nil, err
		}
		author, created := r.commentGitMeta(id, cid)
		out = append(out, CommentMeta{Comment: *c, Author: author, Created: created})
	}
	return out, nil
}

// commentGitMeta derives a comment's author and created date from git history of
// its file path (ADR-0023). Best-effort: returns empties if git has no record.
func (r *Repo) commentGitMeta(id, cid string) (author, created string) {
	full := r.commentPath(id, cid)
	// author: most recent commit touching the file
	if out, err := gitOut(r.Root, "log", "-1", "--format=%an <%ae>", "--", full); err == nil {
		author = strings.TrimSpace(out)
	}
	// created: the commit that ADDED the file (oldest add, if ever re-added)
	if out, err := gitOut(r.Root, "log", "--diff-filter=A", "--format=%aI", "--", full); err == nil {
		if lines := strings.Fields(out); len(lines) > 0 {
			created = lines[len(lines)-1]
		}
	}
	return author, created
}
