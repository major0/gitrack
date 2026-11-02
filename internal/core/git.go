package core

import (
	"fmt"
	"os/exec"
)

// Commit stages and records a commit scoped to the given pathspecs, using the
// system git so the user's own identity, config, and hooks apply (gitrack never
// injects an identity). Only the listed pathspecs are staged and committed
// (`git add -- <paths>` then `git commit -- <paths>`), so changes the user has
// staged OUTSIDE these paths are never swept into the commit and remain staged
// (ADR-0028). Pathspecs are relative to the repo root.
func (r *Repo) Commit(message string, pathspecs ...string) error {
	if len(pathspecs) == 0 {
		return fmt.Errorf("commit requires at least one pathspec")
	}
	addArgs := append([]string{"-C", r.Root, "add", "--"}, pathspecs...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	commitArgs := append([]string{"-C", r.Root, "commit", "-m", message, "--"}, pathspecs...)
	if out, err := exec.Command("git", commitArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}
