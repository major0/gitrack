package web

import (
	"strings"
	"testing"

	"github.com/major0/gitrack/internal/core"
)

func TestBuildThreadNestsReplies(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// a track to reference from a comment body (autolink)
	target, _ := repo.Create(core.NewTrackOpts{Title: "Target Track"})
	slug, _ := repo.GetField(target, "slug")

	comments := []core.CommentMeta{
		{Comment: core.Comment{ID: "aaaa", Body: "top-level, see #" + slug}},
		{Comment: core.Comment{ID: "bbbb", ReplyTo: "aaaa", Body: "a reply"}},
		{Comment: core.Comment{ID: "cccc", Body: "another top-level"}},
		{Comment: core.Comment{ID: "dddd", ReplyTo: "bbbb", Body: "nested reply"}},
	}
	roots := buildThread(repo, "../", comments)

	if len(roots) != 2 {
		t.Fatalf("want 2 root comments, got %d", len(roots))
	}
	// aaaa has one reply bbbb, which has one reply dddd
	var aaaa *commentNode
	for _, r := range roots {
		if r.ID == "aaaa" {
			aaaa = r
		}
	}
	if aaaa == nil || len(aaaa.Replies) != 1 || aaaa.Replies[0].ID != "bbbb" {
		t.Fatalf("aaaa should have reply bbbb, got %+v", aaaa)
	}
	if len(aaaa.Replies[0].Replies) != 1 || aaaa.Replies[0].Replies[0].ID != "dddd" {
		t.Error("bbbb should have nested reply dddd")
	}
	// the #ref in aaaa's body is autolinked to the target track page
	if !strings.Contains(string(aaaa.BodyHTML), `href="../track/`+target+`.html"`) {
		t.Errorf("comment body #ref not autolinked:\n%s", aaaa.BodyHTML)
	}
}

func TestBuildThreadOrphanReplyBecomesRoot(t *testing.T) {
	dir := t.TempDir()
	repo, _ := core.Init(dir)
	// a reply whose parent is absent should still render (as a root), not vanish
	comments := []core.CommentMeta{
		{Comment: core.Comment{ID: "xxxx", ReplyTo: "missing-parent", Body: "orphan"}},
	}
	roots := buildThread(repo, "", comments)
	if len(roots) != 1 || roots[0].ID != "xxxx" {
		t.Errorf("orphan reply should become a root, got %+v", roots)
	}
}
