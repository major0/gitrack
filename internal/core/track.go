package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Relationship directory names (docs/FORMAT.md §4). These are forward links and
// are the source of truth; reverse links are derived into the cache.
var RelationKinds = []string{"depends-on", "child-of", "related", "duplicate-of"}

// Track is a materialized view of one tracker object, read from its authoritative
// files. It is a convenience snapshot; the files on disk remain the truth.
type Track struct {
	ID          string
	Kind        string
	Slug        string
	Title       string
	Description string
	State       string
	Created     string
	Labels      []string
	Assignees   []string
	Relations   map[string][]string // rel kind -> target UUIDs
}

// NewTrackOpts carries the inputs for creating a track.
type NewTrackOpts struct {
	Kind        string
	Title       string
	Description string
	State       string
}

// Create mints a new track with a UUIDv7 identity and writes its authoritative
// fields (docs/FORMAT.md §3, ADR-0002/0006). It returns the new track's ID.
func (r *Repo) Create(opts NewTrackOpts) (string, error) {
	id, err := NewUUIDv7()
	if err != nil {
		return "", err
	}
	kind := opts.Kind
	if kind == "" {
		kind = "task" // cosmetic default (ADR-0006)
	}
	state := opts.State
	if state == "" {
		state = "new"
	}
	dir := r.trackPath(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating track dir: %w", err)
	}
	fields := map[string]string{
		"kind":    kind,
		"slug":    Slugify(opts.Title, id),
		"title":   opts.Title,
		"state":   state,
		"created": time.Now().UTC().Format(time.RFC3339),
	}
	if opts.Description != "" {
		fields["description"] = opts.Description
	}
	for k, v := range fields {
		if err := r.SetField(id, k, v); err != nil {
			return "", err
		}
	}
	return id, nil
}

// GetField reads a single-valued field; a missing field returns "" and no error
// (absence is a valid state in a dumb-data layout).
func (r *Repo) GetField(id, name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(r.trackPath(id), name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading %s/%s: %w", id, name, err)
	}
	return strings.TrimRight(string(b), "\n"), nil
}

// SetField writes a single-valued field as "value\n".
func (r *Repo) SetField(id, name, value string) error {
	path := filepath.Join(r.trackPath(id), name)
	if err := os.WriteFile(path, []byte(value+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing %s/%s: %w", id, name, err)
	}
	return nil
}

// listMulti returns the sorted entry names of a multi-valued attribute
// directory; a missing directory yields an empty slice.
func (r *Repo) listMulti(id, name string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(r.trackPath(id), name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s/%s: %w", id, name, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// Load reads a track's authoritative fields and relationships into a Track.
func (r *Repo) Load(id string) (*Track, error) {
	if _, err := os.Stat(r.trackPath(id)); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNoTrack, id)
	}
	t := &Track{ID: id, Relations: map[string][]string{}}
	for _, f := range []struct {
		name string
		dst  *string
	}{
		{"kind", &t.Kind}, {"slug", &t.Slug}, {"title", &t.Title},
		{"description", &t.Description}, {"state", &t.State}, {"created", &t.Created},
	} {
		v, err := r.GetField(id, f.name)
		if err != nil {
			return nil, err
		}
		*f.dst = v
	}
	var err error
	if t.Labels, err = r.listMulti(id, "labels"); err != nil {
		return nil, err
	}
	if t.Assignees, err = r.listMulti(id, "assignees"); err != nil {
		return nil, err
	}
	for _, rel := range RelationKinds {
		targets, err := r.listMulti(id, rel)
		if err != nil {
			return nil, err
		}
		if len(targets) > 0 {
			t.Relations[rel] = targets
		}
	}
	return t, nil
}

// Slugify derives a repo-unique-ish kebab-case handle from a title, falling back
// to a short id when the title yields nothing usable (ADR-0009/0011). It never
// returns a pure-hex slug, so slugs can't collide with short-id resolution.
func Slugify(title, id string) string {
	var b strings.Builder
	lastDash := true // avoid leading dash
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" || isHex(slug) {
		// fall back to a short-id-derived, guaranteed-non-hex slug
		return "track-" + strings.ReplaceAll(id, "-", "")[:8]
	}
	return slug
}
