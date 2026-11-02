package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fixed lifecycle states (ADR-0019). This set is never configurable.
const (
	StateOpen   = "open"
	StateClosed = "closed"
)

// Built-in mandatory statuses (ADR-0019). Always valid regardless of config;
// user-defined statuses are added under config/statuses/.
const (
	StatusNew      = "new"
	StatusReopened = "reopened"
	StatusDone     = "done"
)

// BuiltinStatuses are the three hard-coded statuses every tracker has.
var BuiltinStatuses = []string{StatusNew, StatusReopened, StatusDone}

// readLines returns the non-empty, comment-stripped lines of a tracker file; a
// missing file yields nil with no error.
func (r *Repo) readLines(rel string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(r.trackerPath(), rel))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", rel, err)
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// Statuses returns every valid status: the three built-ins plus any user-defined
// statuses under config/statuses/ (ADR-0019), de-duplicated.
func (r *Repo) Statuses() ([]string, error) {
	set := map[string]bool{}
	var out []string
	add := func(s string) {
		if !set[s] {
			set[s] = true
			out = append(out, s)
		}
	}
	for _, s := range BuiltinStatuses {
		add(s)
	}
	entries, err := os.ReadDir(filepath.Join(r.trackerPath(), "config", "statuses"))
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("reading statuses config: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			add(e.Name())
		}
	}
	return out, nil
}

// NextStatuses returns the allowed next statuses declared for a status in
// config/statuses/<status>. An absent or empty file means unrestricted (nil).
func (r *Repo) NextStatuses(status string) ([]string, error) {
	return r.readLines(filepath.Join("config", "statuses", status))
}

// Kinds returns the kind allow-list from config/kinds (ADR-0006). It is
// optional: an absent file yields nil, meaning any kind is allowed. The list is
// a guardrail, not enforcement (ADR-0026).
func (r *Repo) Kinds() ([]string, error) {
	return r.readLines(filepath.Join("config", "kinds"))
}

// IsClosed reports whether a track is closed: exactly state == "closed"
// (ADR-0019).
func (r *Repo) IsClosed(id string) (bool, error) {
	state, err := r.GetField(id, "state")
	if err != nil {
		return false, err
	}
	return state == StateClosed, nil
}

// ValidateStatusTransition reports whether moving status from -> to is allowed.
// A built-in status is always a permitted target (new/reopened/done are owned by
// the lifecycle verbs). If "from" declares no next statuses, any move is allowed.
func (r *Repo) ValidateStatusTransition(from, to string) error {
	for _, b := range BuiltinStatuses {
		if to == b {
			return nil
		}
	}
	allowed, err := r.NextStatuses(from)
	if err != nil {
		return err
	}
	if len(allowed) == 0 {
		return nil // unrestricted
	}
	for _, a := range allowed {
		if a == to {
			return nil
		}
	}
	return fmt.Errorf("status transition %q -> %q not allowed (allowed: %s)",
		from, to, strings.Join(allowed, ", "))
}

// IsBuiltinStatus reports whether name is one of the mandatory built-in
// statuses (ADR-0019), which cannot be removed.
func IsBuiltinStatus(name string) bool {
	for _, b := range BuiltinStatuses {
		if name == b {
			return true
		}
	}
	return false
}

// statusesDir is the absolute config/statuses directory.
func (r *Repo) statusesDir() string {
	return filepath.Join(r.trackerPath(), "config", "statuses")
}

// validStatusName reports whether name is an acceptable status identifier:
// kebab-case [a-z0-9-], non-empty, not pure hex (so it never collides with an
// id shape), consistent with slug rules (ADR-0011).
func validStatusName(name string) bool {
	if name == "" || isHex(name) {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// statusExists reports whether a status is valid in this repo (built-in or a
// user-defined file under config/statuses/).
func (r *Repo) statusExists(name string) (bool, error) {
	if IsBuiltinStatus(name) {
		return true, nil
	}
	_, err := os.Stat(filepath.Join(r.statusesDir(), name))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// AddStatus creates a user-defined status with an optional allowed next-set.
// It validates the name and that each next target is an existing status.
func (r *Repo) AddStatus(name string, next []string) error {
	if !validStatusName(name) {
		return fmt.Errorf("invalid status name %q (use kebab-case, not pure hex)", name)
	}
	if exists, err := r.statusExists(name); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("status %q already exists", name)
	}
	return r.writeStatus(name, next)
}

// SetNextStatuses replaces the allowed next-set of an existing status (built-in
// or user-defined). An empty next means unrestricted. Each target must exist.
func (r *Repo) SetNextStatuses(name string, next []string) error {
	if exists, err := r.statusExists(name); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("no such status %q", name)
	}
	return r.writeStatus(name, next)
}

// writeStatus validates targets and writes config/statuses/<name>.
func (r *Repo) writeStatus(name string, next []string) error {
	for _, t := range next {
		if exists, err := r.statusExists(t); err != nil {
			return err
		} else if !exists {
			return fmt.Errorf("next status %q does not exist", t)
		}
	}
	if err := os.MkdirAll(r.statusesDir(), 0o755); err != nil {
		return fmt.Errorf("creating statuses dir: %w", err)
	}
	var body string
	if len(next) > 0 {
		body = strings.Join(next, "\n") + "\n"
	}
	path := filepath.Join(r.statusesDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing status %q: %w", name, err)
	}
	return nil
}

// RemoveStatus deletes a user-defined status. Built-in statuses are refused.
func (r *Repo) RemoveStatus(name string) error {
	if IsBuiltinStatus(name) {
		return fmt.Errorf("%q is a built-in status and cannot be removed", name)
	}
	path := filepath.Join(r.statusesDir(), name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no such status %q", name)
		}
		return fmt.Errorf("removing status %q: %w", name, err)
	}
	return nil
}
