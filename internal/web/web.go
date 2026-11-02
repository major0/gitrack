// Package web generates a self-contained static site from the tracker, to the
// constraints of ADR-0029: the output is a plain directory of HTML that renders
// from file:// or any dumb host with no running server. Core content (track
// list, per-track pages, the Ready/Blocked board) is pure HTML with inline CSS
// and NO network dependency; the dependency digraph is a progressive
// enhancement (Mermaid via CDN) sitting above an always-present HTML fallback.
package web

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"github.com/major0/gitrack/internal/core"
)

// page is the common template data: the site-relative path prefix back to root
// (so links work from any subdirectory and from file://) and the body.
type pageData struct {
	Title string
	Root  string // relative path to site root, e.g. "" or "../"
	Body  template.HTML
}

// Generate writes the static site for repo into outDir (created if needed),
// overwriting prior output. It returns the number of track pages written.
func Generate(repo *core.Repo, outDir string) (int, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, fmt.Errorf("creating out dir: %w", err)
	}
	ids, err := repo.ListIDs()
	if err != nil {
		return 0, err
	}
	tracks := make([]*core.Track, 0, len(ids))
	for _, id := range ids {
		t, err := repo.Load(id)
		if err != nil {
			return 0, err
		}
		tracks = append(tracks, t)
	}
	sort.Slice(tracks, func(i, j int) bool { return tracks[i].ID < tracks[j].ID })

	board, err := repo.ComputeBoard()
	if err != nil {
		return 0, err
	}

	if err := writeIndex(outDir, tracks); err != nil {
		return 0, err
	}
	if err := writeBoard(outDir, board); err != nil {
		return 0, err
	}
	n := 0
	for _, t := range tracks {
		if err := writeTrack(outDir, repo, t); err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}

// trackView is the data for the track page: the track, its description with
// #refs autolinked (ADR-0030), and commit mentions derived from git (ADR-0031).
type trackView struct {
	Track    *core.Track
	DescHTML template.HTML
	Mentions []core.Mention
	Comments []*commentNode
}

// render wraps body in the base layout and writes relPath under outDir.
func render(outDir, relPath string, data pageData) error {
	full := filepath.Join(outDir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("creating %s: %w", relPath, err)
	}
	defer func() { _ = f.Close() }()
	return baseTmpl.Execute(f, data)
}

func short(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}
