package core

import (
	"os/exec"
	"testing"
)

func TestClassifyMention(t *testing.T) {
	cases := []struct {
		msg, handle, want string
	}{
		{"closes #login-flow", "login-flow", "closes"},
		{"Resolves #login-flow after review", "login-flow", "closes"},
		{"fixes #abc", "abc", "closes"},
		{"relates to #login-flow", "login-flow", "relates"},
		{"related to #login-flow", "login-flow", "relates"},
		{"see #login-flow for context", "login-flow", "mentions"},
		{"work on #login-flow", "login-flow", "mentions"},
		{"no reference here", "login-flow", ""},
		// keyword present but for a different handle: still mentions for ours
		{"closes #other and touches #login-flow", "login-flow", "mentions"},
	}
	for _, c := range cases {
		if got := classifyMention(c.msg, c.handle); got != c.want {
			t.Errorf("classifyMention(%q, %q) = %q, want %q", c.msg, c.handle, got, c.want)
		}
	}
}

func TestMentionsFromHistory(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// make it a git repo with an identity
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q", "-b", "trunk")
	run("config", "user.email", "t@t.t")
	run("config", "user.name", "t")

	id, err := repo.Create(NewTrackOpts{Title: "Login Flow"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	slug, _ := repo.GetField(id, "slug") // login-flow
	run("add", "-A")
	run("commit", "-qm", "create track")

	// code commits that mention the track with different verbs
	run("commit", "-q", "--allow-empty", "-m", "Fix parser\n\ncloses #"+slug)
	run("commit", "-q", "--allow-empty", "-m", "WIP, relates to #"+slug)
	run("commit", "-q", "--allow-empty", "-m", "unrelated change")

	m, err := repo.Mentions(id)
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	// expect the create commit (bare #? no) — only the two that reference #slug.
	verbs := map[string]bool{}
	for _, x := range m {
		verbs[x.Verb] = true
	}
	if !verbs["closes"] || !verbs["relates"] {
		t.Errorf("expected a closes and a relates mention, got %+v", m)
	}
	// the unrelated commit must not appear
	for _, x := range m {
		if x.Subject == "unrelated change" {
			t.Error("unrelated commit should not be a mention")
		}
	}
	// a closes mention must NOT have changed the track's state (no auto-close)
	if closed, _ := repo.IsClosed(id); closed {
		t.Error("a 'closes' commit must not auto-close the track")
	}
}
