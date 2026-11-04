package web

import (
	"html/template"
	"sort"

	"github.com/major0/gitrack/internal/core"
)

// releasesSection renders release notes from the tracker (ADR-0041): every track
// of kind "release" becomes a page under releases/, plus a newest-first index.
// Unlike portal/blog it needs no content files; the notes are the track's
// description and the shipped set is its "ships" links.
type releasesSection struct{}

func (s *releasesSection) name() string { return sectionReleases }

// release is a release track with its derived date and explicit tag.
type release struct {
	track *core.Track
	date  string // YYYY-MM-DD, derived from git (ADR-0041)
	tag   string // the explicit "tag" field only; "" when unset
}

// slug returns the page name for a release: its slug, else its id.
func (r release) slug() string {
	if r.track.Slug != "" {
		return r.track.Slug
	}
	return r.track.ID
}

// nav contributes one releases entry, only when a release exists.
func (s *releasesSection) nav(sc *siteContext) []navItem {
	rels, _, err := s.load(sc)
	if err != nil || len(rels) == 0 {
		return nil
	}
	return []navItem{{Label: "releases", Href: "releases/index.html"}}
}

// render writes releases/index.html and one page per release. No releases means
// no output at all (ADR-0026).
func (s *releasesSection) render(sc *siteContext) error {
	rels, byID, err := s.load(sc)
	if err != nil {
		return err
	}
	if len(rels) == 0 {
		return nil
	}
	rows := make([]releaseRow, len(rels))
	for i, rel := range rels {
		rows[i] = releaseRow{
			Slug:     rel.slug(),
			Title:    rel.track.Title,
			Date:     rel.date,
			Released: rel.track.State == core.StateClosed,
			Shipped:  len(rel.track.Relations["ships"]),
		}
		if err := s.writeRelease(sc, rel, byID); err != nil {
			return err
		}
	}
	body, err := renderBody(releasesIndexTmpl, rows)
	if err != nil {
		return err
	}
	return sc.renderPage("releases/index.html", "../", "Releases", body)
}

// writeRelease renders one release page: meta, the notes as Markdown, and the
// shipped tracks.
func (s *releasesSection) writeRelease(sc *siteContext, rel release, byID map[string]*core.Track) error {
	view := releaseView{
		Title:    rel.track.Title,
		Date:     rel.date,
		Tag:      rel.tag,
		Released: rel.track.State == core.StateClosed,
	}
	var css template.CSS
	if rel.track.Description != "" {
		notes, err := sc.markdown("../", rel.track.Description)
		if err != nil {
			return err
		}
		view.NotesHTML, css = notes.HTML, notes.CSS
	}
	for _, id := range rel.track.Relations["ships"] {
		t, ok := byID[id]
		if !ok {
			continue // dangling link: fsck's concern, not a render error
		}
		view.Shipped = append(view.Shipped, shippedItem{
			ID: t.ID, Title: t.Title, State: t.State, Link: sc.issuesEnabled,
		})
	}
	body, err := renderBody(releaseBodyTmpl, view)
	if err != nil {
		return err
	}
	return sc.renderPageCSS("releases/"+rel.slug()+".html", "../", rel.track.Title, body, css)
}

// load returns the release tracks newest-first, plus every track by id so
// shipped links can be resolved without another read. Order is the derived date
// descending (undated last), then slug, so output is deterministic.
func (s *releasesSection) load(sc *siteContext) ([]release, map[string]*core.Track, error) {
	tracks, err := loadTracks(sc.repo)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]*core.Track, len(tracks))
	var rels []release
	for _, t := range tracks {
		byID[t.ID] = t
		if t.Kind != core.KindRelease {
			continue
		}
		date, err := sc.repo.ReleaseDate(t.ID)
		if err != nil {
			return nil, nil, err
		}
		tag, err := sc.repo.GetField(t.ID, "tag")
		if err != nil {
			return nil, nil, err
		}
		rels = append(rels, release{track: t, date: date, tag: tag})
	}
	sort.Slice(rels, func(i, j int) bool {
		di, dj := rels[i].date, rels[j].date
		if (di == "") != (dj == "") {
			return di != "" // dated first
		}
		if di != dj {
			return di > dj // newer first; ISO dates sort lexicographically
		}
		return rels[i].slug() < rels[j].slug()
	})
	return rels, byID, nil
}

// releaseRow is one row of the releases index (exported for html/template).
type releaseRow struct {
	Slug     string
	Title    string
	Date     string
	Released bool
	Shipped  int
}

// shippedItem is one track a release shipped.
type shippedItem struct {
	ID    string
	Title string
	State string
	Link  bool // link to the track page only when that page exists
}

// releaseView is the template data for one release page.
type releaseView struct {
	Title     string
	Date      string
	Tag       string
	Released  bool
	NotesHTML template.HTML
	Shipped   []shippedItem
}
