package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// IsRelationKind reports whether rel is a known relationship type (ADR-0004).
func IsRelationKind(rel string) bool {
	for _, k := range RelationKinds {
		if rel == k {
			return true
		}
	}
	return false
}

// relationPath returns the forward-link file path <id>/<rel>/<target>.
func (r *Repo) relationPath(id, rel, target string) string {
	return filepath.Join(r.trackPath(id), rel, target)
}

// Link adds a forward relationship edge: track `id` --rel--> `target` (ADR-0004).
// Only the forward link is stored; inverse edges are computed (ADR-0005). The
// target must be an existing track. For child-of, a track may have only one
// parent (a convention the CLI enforces, ADR-0004): adding a second is refused.
func (r *Repo) Link(id, rel, target string) error {
	if !IsRelationKind(rel) {
		return fmt.Errorf("unknown relationship %q (have: %v)", rel, RelationKinds)
	}
	if _, err := os.Stat(r.trackPath(id)); err != nil {
		return fmt.Errorf("%w: %s", ErrNoTrack, id)
	}
	if _, err := os.Stat(r.trackPath(target)); err != nil {
		return fmt.Errorf("%w: relationship target %s", ErrNoTrack, target)
	}
	if id == target {
		return fmt.Errorf("a track cannot have a %q relationship to itself", rel)
	}
	if rel == "child-of" {
		existing, err := r.listMulti(id, "child-of")
		if err != nil {
			return err
		}
		for _, p := range existing {
			if p != target {
				return fmt.Errorf("track already has parent %s; a track may have one parent (use 'unlink child-of %s' first)", p, p)
			}
		}
	}
	dir := filepath.Join(r.trackPath(id), rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s dir: %w", rel, err)
	}
	// Empty link file; its name is the target UUID (ADR-0004).
	if err := os.WriteFile(r.relationPath(id, rel, target), nil, 0o644); err != nil {
		return fmt.Errorf("writing %s link: %w", rel, err)
	}
	return nil
}

// Unlink removes a forward relationship edge. A missing edge is an error so the
// caller learns the link was not there.
func (r *Repo) Unlink(id, rel, target string) error {
	if !IsRelationKind(rel) {
		return fmt.Errorf("unknown relationship %q (have: %v)", rel, RelationKinds)
	}
	if err := os.Remove(r.relationPath(id, rel, target)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no %q link from %s to %s", rel, id[:8], target[:8])
		}
		return fmt.Errorf("removing %s link: %w", rel, err)
	}
	return nil
}

// Blockers returns the UUIDs of tracks that must close before `id` is unblocked
// (ADR-0005): everything in id's depends-on/, plus every track that is child-of
// id (an epic waits for its children). It reads the forward links directly (the
// cache is only an accelerator).
func (r *Repo) Blockers(id string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	// A depends-on B  => B blocks A.
	deps, err := r.listMulti(id, "depends-on")
	if err != nil {
		return nil, err
	}
	for _, d := range deps {
		add(d)
	}
	// C child-of id  => C blocks id (parent waits for children).
	ids, err := r.ListIDs()
	if err != nil {
		return nil, err
	}
	for _, other := range ids {
		if other == id {
			continue
		}
		parents, err := r.listMulti(other, "child-of")
		if err != nil {
			return nil, err
		}
		for _, p := range parents {
			if p == id {
				add(other)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// IsUnblocked reports whether `id` has no open blocker (ADR-0005): every
// depends-on target is closed AND every child is closed.
func (r *Repo) IsUnblocked(id string) (bool, error) {
	blockers, err := r.Blockers(id)
	if err != nil {
		return false, err
	}
	for _, b := range blockers {
		closed, err := r.IsClosed(b)
		if err != nil {
			return false, err
		}
		if !closed {
			return false, nil
		}
	}
	return true, nil
}
