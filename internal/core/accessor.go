package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GovernedFields are single-valued fields that must not be written by the
// low-level `set` accessor because lifecycle invariants own them (ADR-0019/0021).
var GovernedFields = map[string]string{
	"state":  "use 'close'/'reopen' to change state",
	"status": "use 'update --status' to move status (it validates the workflow)",
}

// Value is the result of resolving a dotted accessor path. Exactly one of the
// fields is populated, indicated by Kind.
type Value struct {
	Kind   string   // "scalar" | "list" | "object"
	Scalar string   // Kind == scalar
	List   []string // Kind == list (sorted entry names or file content lines)
	Object *Track   // Kind == object (whole track)
}

// Get resolves an accessor of the form "<ref>[.<path>]" against the tree. With
// no path it returns the whole track as an object. A path descends: a field
// name reads that file (scalar); a multi-value attribute name lists the
// directory; a deeper path reads a nested file (e.g. comments.<id>).
func (r *Repo) Get(accessor string) (*Value, error) {
	ref, path, hasPath := strings.Cut(accessor, ".")
	id, err := r.Resolve(ref)
	if err != nil {
		return nil, err
	}
	if !hasPath {
		t, err := r.Load(id)
		if err != nil {
			return nil, err
		}
		return &Value{Kind: "object", Object: t}, nil
	}
	// Descend the remaining dot-separated segments as tree path parts. The
	// storage suffix (e.g. comments' ".md" marking a Markdown body) is a
	// storage detail, never part of the addressable id: when a segment names an
	// attribute whose entries carry a suffix, the next segment is the bare id
	// and the suffix is appended to find the file.
	full := r.trackPath(id)
	segs := strings.Split(path, ".")
	for i := 0; i < len(segs); i++ {
		resolved, err := resolveSegment(full, segs[i])
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrNoField, accessor)
		}
		full = resolved
	}
	fi, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNoField, accessor)
	}
	if fi.IsDir() {
		suffix := attrSuffix(filepath.Base(full))
		entries, err := os.ReadDir(full)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", accessor, err)
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			// list bare ids: strip the attribute's storage suffix (ADR-0022)
			names = append(names, strings.TrimSuffix(e.Name(), suffix))
		}
		sort.Strings(names)
		return &Value{Kind: "list", List: names}, nil
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", accessor, err)
	}
	return &Value{Kind: "scalar", Scalar: strings.TrimRight(string(b), "\n")}, nil
}

// Set writes a free-form single-valued field addressed as "<ref>.<key>". It
// refuses governed fields (state/status), redirecting the caller to the verb
// that enforces the invariant (ADR-0021).
func (r *Repo) Set(accessor, value string) error {
	ref, key, hasKey := strings.Cut(accessor, ".")
	if !hasKey || key == "" {
		return fmt.Errorf("set requires <ref>.<key>, got %q", accessor)
	}
	if strings.Contains(key, ".") {
		return fmt.Errorf("set writes a single field; %q is a nested path", key)
	}
	if hint, governed := GovernedFields[key]; governed {
		return fmt.Errorf("%q is a governed field: %s", key, hint)
	}
	id, err := r.Resolve(ref)
	if err != nil {
		return err
	}
	return r.SetField(id, key, value)
}

// attrStorageSuffix maps a multi-value attribute directory name to the storage
// suffix its entries carry. The suffix marks the on-disk file type (comments are
// Markdown) and is NEVER part of the addressable id (ADR-0022). Attributes not
// listed store bare-named entries.
var attrStorageSuffix = map[string]string{
	"comments": ".md",
}

// attrSuffix returns the storage suffix for an attribute directory, or "".
func attrSuffix(attr string) string { return attrStorageSuffix[attr] }

// resolveSegment returns the child of dir matching seg. It tries the exact name
// first, then — if the parent attribute carries a storage suffix — seg plus that
// suffix (so a bare id addresses <id>.md). It never strips an arbitrary
// extension; the suffix is the attribute's declared one only.
func resolveSegment(dir, seg string) (string, error) {
	exact := filepath.Join(dir, seg)
	if _, err := os.Stat(exact); err == nil {
		return exact, nil
	}
	if suffix := attrSuffix(filepath.Base(dir)); suffix != "" {
		withSuffix := filepath.Join(dir, seg+suffix)
		if _, err := os.Stat(withSuffix); err == nil {
			return withSuffix, nil
		}
	}
	return "", fmt.Errorf("no entry %q in %s", seg, dir)
}

// ErrNoField is returned when an accessor path resolves to no file or directory.
var ErrNoField = fmt.Errorf("no such field or path")
