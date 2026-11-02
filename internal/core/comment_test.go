package core

import (
	"strings"
	"testing"
)

func TestAddAndLoadComment(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Title: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// top-level comment: pure body, no front matter
	cid, err := repo.AddComment(id, "First comment with a #ref and https://example.com", "")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	c, err := repo.LoadComment(id, cid)
	if err != nil {
		t.Fatalf("LoadComment: %v", err)
	}
	if c.ReplyTo != "" {
		t.Errorf("top-level comment should have no reply-to, got %q", c.ReplyTo)
	}
	if !strings.Contains(c.Body, "#ref") || !strings.Contains(c.Body, "https://example.com") {
		t.Errorf("body not preserved verbatim: %q", c.Body)
	}

	// reply: carries reply-to front matter
	rid, err := repo.AddComment(id, "A reply", cid)
	if err != nil {
		t.Fatalf("AddComment reply: %v", err)
	}
	reply, err := repo.LoadComment(id, rid)
	if err != nil {
		t.Fatalf("LoadComment reply: %v", err)
	}
	if reply.ReplyTo != cid {
		t.Errorf("reply.ReplyTo = %q, want %q", reply.ReplyTo, cid)
	}
	if reply.Body != "A reply" {
		t.Errorf("reply.Body = %q", reply.Body)
	}
}

func TestAddCommentRejectsBadReplyTo(t *testing.T) {
	dir := t.TempDir()
	repo, _ := Init(dir)
	id, _ := repo.Create(NewTrackOpts{Title: "x"})
	if _, err := repo.AddComment(id, "body", "no-such-comment-uuid"); err == nil {
		t.Error("reply-to a nonexistent comment should be refused")
	}
}

func TestParseCommentToleratesBothForms(t *testing.T) {
	// body-only
	rt, body := parseComment("just a body\nsecond line\n")
	if rt != "" || body != "just a body\nsecond line" {
		t.Errorf("body-only parse = (%q, %q)", rt, body)
	}
	// front matter + body
	rt, body = parseComment("---\nreply-to: abc\n---\nthe body\n")
	if rt != "abc" || body != "the body" {
		t.Errorf("front-matter parse = (%q, %q)", rt, body)
	}
}

func TestCiteQuote(t *testing.T) {
	got := CiteQuote("line one\n\nline three")
	want := "> line one\n>\n> line three\n"
	if got != want {
		t.Errorf("CiteQuote =\n%q\nwant\n%q", got, want)
	}
}
