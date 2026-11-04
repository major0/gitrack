package web

import (
	"html/template"
	"sort"

	"github.com/major0/gitrack/internal/core"
)

// issuesSection renders the tracker: the track index, the issues (open/closed)
// view, the Ready/Blocked board, and one page per track (ADR-0027/0030/etc.).
// It is the first section of the site (ADR-0036 §1, ADR-0037); the shipped
// tracker web output is produced here.
type issuesSection struct{}

func (s *issuesSection) name() string { return sectionIssues }

// indexHref returns where the track index lives: index.html when issues owns the
// site root (issues-only default, preserving Phase 0 output), else tracks.html
// when another section is home (ADR-0039).
func (s *issuesSection) indexHref(sc *siteContext) string {
	if sc.isHome(sectionIssues) {
		return "index.html"
	}
	return "tracks.html"
}

// nav returns the tracker's three nav entries. When issues owns the site root
// the first entry is index.html labeled "tracks" (byte-identical to pre-ADR-0037
// output); otherwise the track index is tracks.html (ADR-0039).
func (s *issuesSection) nav(sc *siteContext) []navItem {
	return []navItem{
		{Label: "tracks", Href: s.indexHref(sc)},
		{Label: "issues", Href: "issues.html"},
		{Label: "board", Href: "board.html"},
	}
}

// render writes the index, issues, board, and per-track pages.
func (s *issuesSection) render(sc *siteContext) error {
	tracks, err := loadTracks(sc.repo)
	if err != nil {
		return err
	}
	board, err := sc.repo.ComputeBoard()
	if err != nil {
		return err
	}
	if err := s.writeIndex(sc, tracks); err != nil {
		return err
	}
	if err := s.writeIssues(sc, tracks); err != nil {
		return err
	}
	if err := s.writeBoard(sc, board); err != nil {
		return err
	}
	shippedIn := shippedInIndex(sc, tracks)
	for _, t := range tracks {
		if err := s.writeTrack(sc, t, shippedIn[t.ID]); err != nil {
			return err
		}
		sc.trackPages++
	}
	return nil
}

// releaseRef is a link from a shipped track back to a release that shipped it.
type releaseRef struct {
	Title string
	Href  string // relative to track/<id>.html
}

// shippedInIndex inverts the forward "ships" links once (ADR-0041): a map from
// each shipped track id to the releases that shipped it. One pass over the
// tracks, rather than a scan per page. The link targets the release page when
// the releases section is enabled, else the release's own track page.
func shippedInIndex(sc *siteContext, tracks []*core.Track) map[string][]releaseRef {
	idx := map[string][]releaseRef{}
	for _, t := range tracks {
		if t.Kind != core.KindRelease {
			continue
		}
		href := t.ID + ".html"
		if sc.releasesEnabled {
			name := t.Slug
			if name == "" {
				name = t.ID
			}
			href = "../releases/" + name + ".html"
		}
		for _, target := range t.Relations["ships"] {
			idx[target] = append(idx[target], releaseRef{Title: t.Title, Href: href})
		}
	}
	return idx
}

// loadTracks loads every track, sorted by id for deterministic output.
func loadTracks(repo *core.Repo) ([]*core.Track, error) {
	ids, err := repo.ListIDs()
	if err != nil {
		return nil, err
	}
	tracks := make([]*core.Track, 0, len(ids))
	for _, id := range ids {
		t, err := repo.Load(id)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	sort.Slice(tracks, func(i, j int) bool { return tracks[i].ID < tracks[j].ID })
	return tracks, nil
}

// trackView is the data for a track page: the track, its description with
// #refs autolinked (ADR-0030), commit mentions derived from git (ADR-0031), and
// the threaded comments.
type trackView struct {
	Track     *core.Track
	DescHTML  template.HTML
	Mentions  []core.Mention
	Comments  []*commentNode
	ShippedIn []releaseRef
}

func (s *issuesSection) writeIndex(sc *siteContext, tracks []*core.Track) error {
	body, err := renderBody(indexBodyTmpl, tracks)
	if err != nil {
		return err
	}
	return sc.renderPage(s.indexHref(sc), "", "gitrack", body)
}

func (s *issuesSection) writeIssues(sc *siteContext, tracks []*core.Track) error {
	var open, closed []*core.Track
	for _, t := range tracks {
		if t.State == core.StateClosed {
			closed = append(closed, t)
		} else {
			open = append(open, t)
		}
	}
	view := struct {
		Open   []*core.Track
		Closed []*core.Track
	}{open, closed}
	body, err := renderBody(issuesBodyTmpl, view)
	if err != nil {
		return err
	}
	return sc.renderPage("issues.html", "", "gitrack issues", body)
}

func (s *issuesSection) writeBoard(sc *siteContext, board *core.Board) error {
	var ready, blocked []core.BoardEntry
	for _, e := range board.Entries {
		if !e.Open {
			continue
		}
		if e.Ready() {
			ready = append(ready, e)
		} else {
			blocked = append(blocked, e)
		}
	}
	view := boardView{Ready: ready, Blocked: blocked, Mermaid: board.Mermaid()}
	body, err := renderBody(boardBodyTmpl, view)
	if err != nil {
		return err
	}
	return sc.renderPage("board.html", "", "gitrack board", body)
}

func (s *issuesSection) writeTrack(sc *siteContext, t *core.Track, shippedIn []releaseRef) error {
	// Track pages live in track/<id>.html, one level below root, so refs link
	// back up via "../" (ADR-0030).
	mentions, err := sc.repo.Mentions(t.ID)
	if err != nil {
		return err
	}
	comments, err := sc.repo.ListComments(t.ID)
	if err != nil {
		return err
	}
	view := trackView{
		Track:     t,
		DescHTML:  autolink(sc.repo, "../", t.Description),
		Mentions:  mentions,
		Comments:  buildThread(sc.repo, "../", comments),
		ShippedIn: shippedIn,
	}
	body, err := renderBody(trackBodyTmpl, view)
	if err != nil {
		return err
	}
	return sc.renderPage("track/"+t.ID+".html", "../", t.Title, body)
}
