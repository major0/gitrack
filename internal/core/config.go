package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readLines returns the non-empty, comment-stripped lines of a config file; a
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

// ClosedStates returns the set of states that count as closed
// (config/states.closed). A track whose state is in this set is closed.
func (r *Repo) ClosedStates() (map[string]bool, error) {
	lines, err := r.readLines(filepath.Join("config", "states.closed"))
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(lines))
	for _, s := range lines {
		set[s] = true
	}
	return set, nil
}

// IsClosed reports whether a track is in a closed state.
func (r *Repo) IsClosed(id string) (bool, error) {
	closed, err := r.ClosedStates()
	if err != nil {
		return false, err
	}
	state, err := r.GetField(id, "state")
	if err != nil {
		return false, err
	}
	return closed[state], nil
}
