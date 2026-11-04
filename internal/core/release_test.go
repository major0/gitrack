package core

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// gitIn runs git in dir with extra environment, failing the test on error.
func gitIn(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// newGitRepo makes an initialized tracker inside a real git repo.
func newGitRepo(t *testing.T) (*Repo, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	gitIn(t, dir, nil, "init", "-q", "-b", "trunk")
	gitIn(t, dir, nil, "config", "user.email", "t@t.t")
	gitIn(t, dir, nil, "config", "user.name", "t")
	return repo, dir
}

func TestReleaseTagFallsBackToTitle(t *testing.T) {
	repo, err := Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	id, err := repo.Create(NewTrackOpts{Kind: KindRelease, Title: "v1.0.0"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, _ := repo.ReleaseTag(id); got != "v1.0.0" {
		t.Errorf("ReleaseTag = %q, want the title v1.0.0 when no tag is set", got)
	}
	if err := repo.SetField(id, "tag", "release-1"); err != nil {
		t.Fatalf("SetField: %v", err)
	}
	if got, _ := repo.ReleaseTag(id); got != "release-1" {
		t.Errorf("ReleaseTag = %q, want the explicit tag release-1", got)
	}
}

func TestReleaseDateFromTag(t *testing.T) {
	repo, dir := newGitRepo(t)
	id, err := repo.Create(NewTrackOpts{Kind: KindRelease, Title: "v1.0.0"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	gitIn(t, dir, nil, "add", "-A")
	when := "2025-03-04T10:00:00Z"
	gitIn(t, dir, []string{"GIT_COMMITTER_DATE=" + when, "GIT_AUTHOR_DATE=" + when},
		"commit", "-qm", "release commit")
	gitIn(t, dir, nil, "tag", "v1.0.0")

	got, err := repo.ReleaseDate(id)
	if err != nil {
		t.Fatalf("ReleaseDate: %v", err)
	}
	if got != "2025-03-04" {
		t.Errorf("ReleaseDate = %q, want 2025-03-04 from the tag's commit", got)
	}
}

func TestReleaseDateFallsBackToCreated(t *testing.T) {
	repo, _ := newGitRepo(t)
	id, _ := repo.Create(NewTrackOpts{Kind: KindRelease, Title: "v9.9.9"})
	created, _ := repo.GetField(id, "created")
	ts, err := time.Parse(time.RFC3339, created)
	if err != nil {
		t.Fatalf("created %q: %v", created, err)
	}
	got, err := repo.ReleaseDate(id)
	if err != nil {
		t.Fatalf("ReleaseDate: %v", err)
	}
	if want := ts.UTC().Format("2006-01-02"); got != want {
		t.Errorf("ReleaseDate = %q, want created date %q when the tag does not exist", got, want)
	}
}

func TestReleaseDateOptionLikeTagIsNotAnOption(t *testing.T) {
	repo, dir := newGitRepo(t)
	id, _ := repo.Create(NewTrackOpts{Kind: KindRelease, Title: "v1"})
	// a tag value that looks like a git option must be treated as a ref name,
	// so it simply does not resolve and the date falls back to "created"
	if err := repo.SetField(id, "tag", "--output=pwned"); err != nil {
		t.Fatalf("SetField: %v", err)
	}
	got, err := repo.ReleaseDate(id)
	if err != nil {
		t.Fatalf("ReleaseDate: %v", err)
	}
	if got == "" {
		t.Errorf("expected the created-date fallback, got empty")
	}
	if _, statErr := os.Stat(dir + "/pwned"); !os.IsNotExist(statErr) {
		t.Errorf("option-like tag must not be interpreted by git (stat err=%v)", statErr)
	}
}

func TestShipsLinkHasNoSchedulingEffect(t *testing.T) {
	repo, err := Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	rel, _ := repo.Create(NewTrackOpts{Kind: KindRelease, Title: "v1.0.0"})
	issue, _ := repo.Create(NewTrackOpts{Title: "Login"})
	if err := repo.Link(rel, "ships", issue); err != nil {
		t.Fatalf("Link ships: %v", err)
	}
	tr, err := repo.Load(rel)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := tr.Relations["ships"]; len(got) != 1 || got[0] != issue {
		t.Errorf("ships = %v, want [%s]", got, issue)
	}
	// shipping an OPEN track must not block the release, nor block the track
	if ok, _ := repo.IsUnblocked(rel); !ok {
		t.Errorf("a release must not be blocked by a track it ships")
	}
	if ok, _ := repo.IsUnblocked(issue); !ok {
		t.Errorf("a shipped track must not be blocked by the release")
	}
	if err := repo.Unlink(rel, "ships", issue); err != nil {
		t.Fatalf("Unlink ships: %v", err)
	}
	tr, _ = repo.Load(rel)
	if len(tr.Relations["ships"]) != 0 {
		t.Errorf("ships should be empty after unlink, got %v", tr.Relations["ships"])
	}
	if !strings.Contains(strings.Join(RelationKinds, ","), "ships") {
		t.Errorf("ships missing from RelationKinds: %v", RelationKinds)
	}
}
