package core

import (
	"fmt"
	"os/exec"
)

// Commit stages the tracker path and records a commit, using the system git so
// the user's own identity, config, and hooks apply (gitrack never injects an
// identity). pathspec is staged relative to the repo root; message is the commit
// subject. It is used by commands that opt into committing (e.g. init); most
// mutating commands leave staging to the user so tracker changes ride along with
// the related code change.
func (r *Repo) Commit(pathspec, message string) error {
	// Stage only the given pathspec (not `git add -A`) so we never sweep up
	// unrelated working-tree changes.
	add := exec.Command("git", "-C", r.Root, "add", "--", pathspec)
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add %s: %w: %s", pathspec, err, out)
	}
	commit := exec.Command("git", "-C", r.Root, "commit", "-m", message, "--", pathspec)
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}
