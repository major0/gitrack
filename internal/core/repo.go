// Package core reads and writes the authoritative gitrack tracker tree as
// specified in docs/FORMAT.md. It is deliberately "dumb": a single-valued field
// is a file whose name is the key and whose content is the value; a multi-valued
// attribute is a directory of entries. Derived views (cache, fsck) live in
// sibling packages and are computed from what this package reads.
package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Dir is the tracker subdirectory within a repository working tree.
const Dir = ".gitrack"

// ErrNotTracker is returned when a path has no initialized .gitrack tree.
var ErrNotTracker = errors.New("not a gitrack tracker (run 'git track init')")

// ErrNoTrack is returned when a track id resolves to no object.
var ErrNoTrack = errors.New("no such track")

// ErrAmbiguous is returned when a short id prefix matches more than one track.
var ErrAmbiguous = errors.New("ambiguous track id")

// Repo is a handle to an initialized tracker tree rooted at a working directory.
type Repo struct {
	// Root is the absolute path to the directory that contains .gitrack.
	Root string
}

// root returns the absolute .gitrack directory path.
func (r *Repo) trackerPath() string { return filepath.Join(r.Root, Dir) }

// objectsGlobRoot returns the .gitrack directory holding per-track object dirs.
// Tracks live directly under .gitrack/<uuid>/ (docs/FORMAT.md §3).
func (r *Repo) objectsRoot() string { return r.trackerPath() }

// Open finds the nearest initialized tracker at or above start and returns a
// Repo for it. It walks upward like git does for a work tree.
func Open(start string) (*Repo, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", start, err)
	}
	for {
		if fi, err := os.Stat(filepath.Join(abs, Dir)); err == nil && fi.IsDir() {
			return &Repo{Root: abs}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil, ErrNotTracker
		}
		abs = parent
	}
}

// Init creates a new tracker tree under root with default configuration and the
// cache .gitignore (ADR-0014). It fails if one already exists.
func Init(root string) (*Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", root, err)
	}
	tp := filepath.Join(abs, Dir)
	if _, err := os.Stat(tp); err == nil {
		return nil, fmt.Errorf("%s already exists", tp)
	}
	if err := os.MkdirAll(filepath.Join(tp, "config", "statuses"), 0o755); err != nil {
		return nil, fmt.Errorf("creating config dir: %w", err)
	}
	// git-ignore the derived cache; keep everything else tracked (ADR-0014).
	if err := os.WriteFile(filepath.Join(tp, ".gitignore"), []byte("/cache/\n"), 0o644); err != nil {
		return nil, fmt.Errorf("writing .gitignore: %w", err)
	}
	// Kinds (cosmetic, ADR-0006). Statuses follow ADR-0019: new/reopened/done are
	// built in; a starter workflow is seeded as per-status next-status files so a
	// fresh tracker demonstrates the model (each file = that status's transitions).
	// state (open/closed) is fixed and needs no config.
	writes := map[string]string{
		filepath.Join("config", "kinds"):                   "task\nbug\nnote\nepic\nproject\n",
		filepath.Join("config", "statuses", "new"):         "triaged\nin-progress\ndone\n",
		filepath.Join("config", "statuses", "triaged"):     "in-progress\ndone\n",
		filepath.Join("config", "statuses", "in-progress"): "in-review\ndone\n",
		filepath.Join("config", "statuses", "in-review"):   "in-progress\ndone\n",
		filepath.Join("config", "statuses", "reopened"):    "in-progress\ndone\n",
	}
	for rel, content := range writes {
		if err := os.WriteFile(filepath.Join(tp, rel), []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", rel, err)
		}
	}
	return &Repo{Root: abs}, nil
}

// trackPath returns the directory of the track with the given full UUID.
func (r *Repo) trackPath(id string) string {
	return filepath.Join(r.objectsRoot(), id)
}

// ListIDs returns the full UUIDs of every track, sorted (UUIDv7 sorts
// chronologically).
func (r *Repo) ListIDs() ([]string, error) {
	entries, err := os.ReadDir(r.objectsRoot())
	if err != nil {
		return nil, fmt.Errorf("reading tracker: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() && IsUUID(e.Name()) {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// Resolve turns a reference handle into a full track UUID. It accepts a full
// UUID, a unique hex short-id prefix (git-short-hash style), or a slug
// (docs/FORMAT.md §8.1). Ambiguous short ids return ErrAmbiguous.
func (r *Repo) Resolve(handle string) (string, error) {
	handle = strings.TrimPrefix(handle, "#")
	if IsUUID(handle) {
		if _, err := os.Stat(r.trackPath(handle)); err == nil {
			return handle, nil
		}
		return "", fmt.Errorf("%w: %s", ErrNoTrack, handle)
	}
	ids, err := r.ListIDs()
	if err != nil {
		return "", err
	}
	// Try slug match first (a slug may not be pure hex, so no overlap with
	// short-id resolution — ADR-0011).
	if !isHex(handle) {
		var match string
		for _, id := range ids {
			slug, _ := r.GetField(id, "slug")
			if slug == handle {
				if match != "" {
					return "", fmt.Errorf("%w: slug %q", ErrAmbiguous, handle)
				}
				match = id
			}
		}
		if match != "" {
			return match, nil
		}
		return "", fmt.Errorf("%w: %s", ErrNoTrack, handle)
	}
	// Hex: treat as a short-id prefix.
	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(strings.ReplaceAll(id, "-", ""), handle) ||
			strings.HasPrefix(id, handle) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %s", ErrNoTrack, handle)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %q matches %d tracks", ErrAmbiguous, handle, len(matches))
	}
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isHexDigit(r) {
			return false
		}
	}
	return true
}
