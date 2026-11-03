package web

import (
	"bytes"
	"fmt"
	"html/template"
)

// renderBody executes a body template to HTML for embedding in the theme shell.
// Local tracker data is rendered as template.HTML; authored text is passed
// through autolink first (escape-then-linkify, ADR-0030) before reaching here.
func renderBody(t *template.Template, data any) (template.HTML, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering body: %w", err)
	}
	return template.HTML(buf.String()), nil //nolint:gosec // local tracker data, not external input
}
