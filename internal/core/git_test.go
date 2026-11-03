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

func TestCommitStagedBundlesIndex(t *testing.T) {
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
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

	id, err := repo.Create(NewTrackOpts{Title: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// user stages a code file; gitrack closes the track and stages the tracker
	if err := exec.Command("git", "-C", dir, "add", "-A").Run(); err != nil {
		t.Fatalf("stage initial: %v", err)
	}
	if err := repo.CommitStaged("init"); err != nil {
		t.Fatalf("CommitStaged init: %v", err)
	}

	// now: stage a code file, close the track, stage tracker, commit together
	if err := exec.Command("git", "-C", dir, "commit", "--allow-empty", "-qm", "base").Run(); err != nil {
		t.Fatalf("base: %v", err)
	}
	// write a code file and stage it
	if err := exec.Command("git", "-C", dir, "init").Run(); err == nil { // no-op guard
		_ = err
	}
	// create a staged file
	f := dir + "/fix.go"
	if err := exec.Command("sh", "-c", "echo fixed > "+f).Run(); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("add", "fix.go")
	if err := repo.Close(id, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := repo.StageTracker(); err != nil {
		t.Fatalf("StageTracker: %v", err)
	}
	if err := repo.CommitStaged("fix and resolve"); err != nil {
		t.Fatalf("CommitStaged: %v", err)
	}

	// the HEAD commit must contain both fix.go and the track state change
	if out, err := exec.Command("git", "-C", dir, "cat-file", "-e", "HEAD:fix.go").CombinedOutput(); err != nil {
		t.Errorf("fix.go not in commit: %s", out)
	}
	if out, err := exec.Command("git", "-C", dir, "cat-file", "-e", "HEAD:.gitrack/"+id+"/state").CombinedOutput(); err != nil {
		t.Errorf("track state not in commit: %s", out)
	}
	if closed, _ := repo.IsClosed(id); !closed {
		t.Error("track should be closed")
	}
}

func TestParseDirectives(t *testing.T) {
	msg := "Implement login\n\ncloses #login-flow\nrelates to #epic-auth\nsee #notes-doc"
	ds := ParseDirectives(msg)
	got := map[string]string{}
	for _, d := range ds {
		got[d.Handle] = d.Verb
	}
	if got["login-flow"] != "closes" {
		t.Errorf("login-flow verb = %q, want closes", got["login-flow"])
	}
	if got["epic-auth"] != "relates" {
		t.Errorf("epic-auth verb = %q, want relates", got["epic-auth"])
	}
	if got["notes-doc"] != "mentions" {
		t.Errorf("notes-doc verb = %q, want mentions", got["notes-doc"])
	}
	// strongest verb wins when a handle appears twice
	msg2 := "touches #x and then closes #x"
	for _, d := range ParseDirectives(msg2) {
		if d.Handle == "x" && d.Verb != "closes" {
			t.Errorf("x verb = %q, want closes (strongest)", d.Verb)
		}
	}
	// no directives in a plain message
	if len(ParseDirectives("just a normal message")) != 0 {
		t.Error("plain message should yield no directives")
	}
}

// TestClosingKeywordLists covers the GitLab-superset list behavior (ADR-0035):
// one keyword governs the whole list of refs that follows it.
func TestClosingKeywordLists(t *testing.T) {
	verbOf := func(message string) map[string]string {
		m := map[string]string{}
		for _, d := range ParseDirectives(message) {
			m[d.Handle] = d.Verb
		}
		return m
	}
	cases := []struct {
		name    string
		message string
		closes  []string // handles expected to be "closes"
		other   map[string]string
	}{
		{"space list", "Closes #a #b #c", []string{"a", "b", "c"}, nil},
		{"comma list", "Resolves #a, #b, #c", []string{"a", "b", "c"}, nil},
		{"comma-and list", "Closes #a, #b, and #c", []string{"a", "b", "c"}, nil},
		{"keyword before each", "fixes #d, resolve #e", []string{"d", "e"}, nil},
		{"footer caps", "feat: thing\n\nCloses #f #g", []string{"f", "g"}, nil},
		{"implement keyword", "implements #h", []string{"h"}, nil},
		{"run ends at a word", "closes #i then wip #j",
			[]string{"i"}, map[string]string{"j": "mentions"}},
		{"issue filler", "Closes issues #k and #l", []string{"k", "l"}, nil},
	}
	for _, c := range cases {
		got := verbOf(c.message)
		for _, h := range c.closes {
			if got[h] != "closes" {
				t.Errorf("[%s] %q: handle %q verb = %q, want closes", c.name, c.message, h, got[h])
			}
		}
		for h, want := range c.other {
			if got[h] != want {
				t.Errorf("[%s] %q: handle %q verb = %q, want %q", c.name, c.message, h, got[h], want)
			}
		}
	}
}
