package web

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/major0/gitrack/internal/core"
)

// renderBody executes a body template to HTML for embedding in the base layout.
func renderBody(t *template.Template, data any) (template.HTML, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering body: %w", err)
	}
	return template.HTML(buf.String()), nil //nolint:gosec // local tracker data, not external input
}

func writeIndex(outDir string, tracks []*core.Track) error {
	body, err := renderBody(indexBodyTmpl, tracks)
	if err != nil {
		return err
	}
	return render(outDir, "index.html", pageData{Title: "gitrack", Root: "", Body: body})
}

func writeTrack(outDir string, t *core.Track) error {
	body, err := renderBody(trackBodyTmpl, t)
	if err != nil {
		return err
	}
	// track pages live in track/<id>.html, one level below root.
	return render(outDir, "track/"+t.ID+".html",
		pageData{Title: t.Title, Root: "../", Body: body})
}

func writeBoard(outDir string, board *core.Board) error {
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
	return render(outDir, "board.html", pageData{Title: "gitrack board", Root: "", Body: body})
}
