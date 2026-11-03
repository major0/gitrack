package web

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

// theme is the shared page shell (layout, nav, CSS) for the whole site
// (ADR-0036 §2): a named set of templates + assets that every section renders
// its body into. Restyling the site is changing the theme in one place. The
// built-in default theme ships inline so the output works offline from file://
// (ADR-0029).
type theme struct {
	// name is the theme identifier; "" / "default" is the built-in theme.
	name string
	// shell is the layout template executed with pageData.
	shell *template.Template
}

// newTheme resolves a theme by name. Phase 0 ships only the built-in default
// theme; an unknown/empty name falls back to it. A repo-provided theme
// directory is a documented follow-on (ADR-0036 §2), resolved here later.
func newTheme(name string) (*theme, error) {
	switch name {
	case "", "default":
		return &theme{name: "default", shell: baseTmpl}, nil
	default:
		// No theme-override resolution yet; fall back to the built-in so a
		// stray theme name never breaks generation (guardrails, ADR-0026).
		return &theme{name: "default", shell: baseTmpl}, nil
	}
}

// render wraps data.Body in the theme shell and writes relPath under outDir,
// creating parent directories as needed.
func (th *theme) render(outDir, relPath string, data pageData) error {
	full := filepath.Join(outDir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("creating %s: %w", relPath, err)
	}
	defer func() { _ = f.Close() }()
	return th.shell.Execute(f, data)
}
