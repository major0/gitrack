package core

import (
	"os/exec"
	"strings"
	"time"
)

// KindRelease is the track kind that marks a release (ADR-0041). The kind is
// cosmetic (ADR-0006); generators list tracks of this kind as releases.
const KindRelease = "release"

// ReleaseTag returns the git tag a release corresponds to: its "tag" field, or
// its title when no tag is set (ADR-0041).
func (r *Repo) ReleaseTag(id string) (string, error) {
	tag, err := r.GetField(id, "tag")
	if err != nil {
		return "", err
	}
	if tag != "" {
		return tag, nil
	}
	return r.GetField(id, "title")
}

// ReleaseDate returns a release's date as YYYY-MM-DD. It is derived from git,
// not stored (ADR-0041): the committer date of the tag's commit when the tag
// exists, else the date part of the track's "created" timestamp, else "".
func (r *Repo) ReleaseDate(id string) (string, error) {
	tag, err := r.ReleaseTag(id)
	if err != nil {
		return "", err
	}
	if d, ok := r.tagDate(tag); ok {
		return d, nil
	}
	created, err := r.GetField(id, "created")
	if err != nil {
		return "", err
	}
	if t, perr := time.Parse(time.RFC3339, created); perr == nil {
		return t.UTC().Format("2006-01-02"), nil
	}
	return "", nil
}

// tagDate reads the committer date of a tag's commit. The "refs/tags/" prefix
// means the value can never be parsed as a git option, whatever the tag text.
// ok is false when the tag does not exist or git cannot resolve it.
func (r *Repo) tagDate(tag string) (string, bool) {
	if tag == "" {
		return "", false
	}
	out, err := exec.Command("git", "-C", r.Root, "log", "-1",
		"--format=%cI", "refs/tags/"+tag+"^{commit}", "--").Output()
	if err != nil {
		return "", false
	}
	iso := strings.TrimSpace(string(out))
	t, perr := time.Parse(time.RFC3339, iso)
	if perr != nil {
		return "", false
	}
	return t.UTC().Format("2006-01-02"), true
}
