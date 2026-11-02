// Package cache builds and verifies the git-ignored derived cache under
// .gitrack/cache/ (ADR-0014). The cache is a pure function of the authoritative
// tracker tree: state and type indexes plus reverse relationship links. It is
// never a source of truth — fsck rebuilds it from the authoritative objects.
package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/major0/gitrack/internal/core"
)

// relPath is the cache directory under the tracker root.
const relPath = "cache"

// Report summarizes a verification pass: what the cache currently holds versus
// what the authoritative data requires.
type Report struct {
	Missing  []string // index entries that should exist but do not
	Extra    []string // index entries that exist but should not
	Dangling []string // forward links pointing at nonexistent tracks
	Warnings []string // guardrail divergences (ADR-0026): advisory, not auto-fixed
}

// Clean reports whether the cache matches the authoritative data AND no
// guardrail warnings were found.
func (r Report) Clean() bool {
	return len(r.Missing) == 0 && len(r.Extra) == 0 &&
		len(r.Dangling) == 0 && len(r.Warnings) == 0
}

// cacheRoot returns the absolute cache directory for a repo.
func cacheRoot(repo *core.Repo) string {
	return filepath.Join(repo.Root, core.Dir, relPath)
}

// desired computes the full set of cache entry paths (relative to the cache
// root) implied by the authoritative tree, plus any dangling forward links.
func desired(repo *core.Repo) (map[string]bool, []string, error) {
	ids, err := repo.ListIDs()
	if err != nil {
		return nil, nil, err
	}
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
		known[id] = true
	}

	want := map[string]bool{}
	var dangling []string
	for _, id := range ids {
		t, err := repo.Load(id)
		if err != nil {
			return nil, nil, err
		}
		if t.State != "" {
			want[filepath.Join("state", t.State, id)] = true
		}
		if t.Status != "" {
			want[filepath.Join("status", t.Status, id)] = true
		}
		if t.Kind != "" {
			want[filepath.Join("type", t.Kind, id)] = true
		}
		// reverse relationship links: refs-in/<target>/<rel>/<referrer>
		for rel, targets := range t.Relations {
			for _, target := range targets {
				if !known[target] {
					dangling = append(dangling, fmt.Sprintf("%s %s -> %s", id, rel, target))
					continue
				}
				want[filepath.Join("refs-in", target, rel, id)] = true
			}
		}
	}
	sort.Strings(dangling)
	return want, dangling, nil
}

// have returns the set of entry paths currently present in the cache (relative
// to the cache root), considering only leaf files.
func have(repo *core.Repo) (map[string]bool, error) {
	root := cacheRoot(repo)
	present := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		present[rel] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking cache: %w", err)
	}
	return present, nil
}

// Verify compares the cache against the authoritative data without changing it.
func Verify(repo *core.Repo) (Report, error) {
	want, dangling, err := desired(repo)
	if err != nil {
		return Report{}, err
	}
	present, err := have(repo)
	if err != nil {
		return Report{}, err
	}
	var rep Report
	rep.Dangling = dangling
	for p := range want {
		if !present[p] {
			rep.Missing = append(rep.Missing, p)
		}
	}
	for p := range present {
		if !want[p] {
			rep.Extra = append(rep.Extra, p)
		}
	}
	sort.Strings(rep.Missing)
	sort.Strings(rep.Extra)
	warn, err := guardrailWarnings(repo)
	if err != nil {
		return Report{}, err
	}
	rep.Warnings = warn
	return rep, nil
}

// guardrailWarnings reports where authoritative track data diverges from the
// declared config guardrails (ADR-0026). These are advisory: the data is valid
// and never rewritten; fsck surfaces the divergence for a human to resolve.
func guardrailWarnings(repo *core.Repo) ([]string, error) {
	ids, err := repo.ListIDs()
	if err != nil {
		return nil, err
	}
	statuses, err := repo.Statuses()
	if err != nil {
		return nil, err
	}
	statusOK := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		statusOK[s] = true
	}
	// config/kinds is an optional allow-list; empty means "any kind allowed".
	kinds, err := repo.Kinds()
	if err != nil {
		return nil, err
	}
	kindOK := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		kindOK[k] = true
	}

	var warn []string
	for _, id := range ids {
		t, err := repo.Load(id)
		if err != nil {
			return nil, err
		}
		short := id[:8]
		if t.State != core.StateOpen && t.State != core.StateClosed {
			warn = append(warn, fmt.Sprintf("%s: state %q is not open/closed", short, t.State))
		}
		if t.Status != "" && !statusOK[t.Status] {
			warn = append(warn, fmt.Sprintf("%s: status %q is not a defined status", short, t.Status))
		}
		if len(kindOK) > 0 && t.Kind != "" && !kindOK[t.Kind] {
			warn = append(warn, fmt.Sprintf("%s: kind %q is not in config/kinds", short, t.Kind))
		}
	}
	sort.Strings(warn)
	return warn, nil
}

// Rebuild regenerates the cache from scratch: it removes the existing cache tree
// and writes exactly the entries implied by the authoritative data. This is
// fsck --write (ADR-0014): a fresh clone or a hand-edited tree is made correct
// by one call, and it can never desync because it is a full regeneration.
func Rebuild(repo *core.Repo) (Report, error) {
	want, dangling, err := desired(repo)
	if err != nil {
		return Report{}, err
	}
	root := cacheRoot(repo)
	if err := os.RemoveAll(root); err != nil {
		return Report{}, fmt.Errorf("clearing cache: %w", err)
	}
	for rel := range want {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return Report{}, fmt.Errorf("creating cache dir: %w", err)
		}
		// Empty marker file; the entry's existence is the datum (ADR-0014).
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			return Report{}, fmt.Errorf("writing cache entry: %w", err)
		}
	}
	// A rebuild fixes the cache, but guardrail divergences are authored-data
	// facts (ADR-0026) a rebuild must NOT silently fix — report them.
	warn, err := guardrailWarnings(repo)
	if err != nil {
		return Report{}, err
	}
	return Report{Dangling: dangling, Warnings: warn}, nil
}
