package web

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// builtinThemes holds the themes that ship in the binary, one directory each:
// themes/<name>/theme.css and anything it needs. They are ordinary files, so a
// built-in theme is written exactly like one a project supplies (ADR-0048).
//
//go:embed themes
var builtinThemes embed.FS

const (
	// defaultThemeName is the theme used when none is configured or the configured
	// name matches nothing.
	defaultThemeName = "default"
	// themeConfigDir holds a project's own themes: .gitrack/config/site/themes/<name>/.
	themeConfigDir = siteConfigDir + "/themes"
	// themeStylesheet is the one file every theme must provide.
	themeStylesheet = "theme.css"
	// themeScript is an optional script a theme may provide.
	themeScript = "theme.js"
	// assetsDir is where every generated stylesheet and script is written.
	assetsDir = "assets"
	// themeAssetsDir is where a theme's files are written, keeping their layout so
	// a stylesheet can refer to its fonts and images by relative path.
	themeAssetsDir = assetsDir + "/theme"

	// maxThemeFiles and maxThemeBytes bound a project's theme directory so a stray
	// large tree cannot be copied into every site by accident.
	maxThemeFiles = 200
	maxThemeBytes = 8 << 20
)

// themeNameRe is the shape of a usable theme name: it becomes a directory name,
// so it may not hold a path separator or start with a dot.
var themeNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// theme is a named set of files: a stylesheet, an optional script, and whatever
// assets they use. It is the whole of a site's presentation; the HTML a page
// carries is the same whichever theme is chosen (ADR-0048).
type theme struct {
	name string
	// files maps a slash-separated path relative to the theme directory to its
	// content.
	files map[string][]byte
}

// hasScript reports whether the theme provides a script of its own.
func (t *theme) hasScript() bool {
	_, ok := t.files[themeScript]
	return ok
}

// loadTheme resolves a theme by name. A directory of that name in the project's
// themes wins over a built-in one; a name that matches neither, or that cannot be
// a directory name, falls back to the default theme so a typo never breaks
// generation (ADR-0026). A project theme directory without a stylesheet is an
// error: the directory is there, so the mistake is real and worth naming.
func loadTheme(repo *core.Repo, name string) (*theme, error) {
	name = strings.TrimSpace(name)
	if name == "" || !themeNameRe.MatchString(name) {
		name = defaultThemeName
	}

	userDir := filepath.Join(repo.Root, core.Dir, themeConfigDir, name)
	if st, err := os.Stat(userDir); err == nil && st.IsDir() {
		files, err := readThemeDir(userDir)
		if err != nil {
			return nil, fmt.Errorf("theme %q: %w", name, err)
		}
		if _, ok := files[themeStylesheet]; !ok {
			return nil, fmt.Errorf("theme %q: %s/%s/%s is required", name, themeConfigDir, name, themeStylesheet)
		}
		return &theme{name: name, files: files}, nil
	}

	if files, err := readBuiltinTheme(name); err == nil {
		return &theme{name: name, files: files}, nil
	}
	files, err := readBuiltinTheme(defaultThemeName)
	if err != nil {
		return nil, fmt.Errorf("built-in default theme: %w", err)
	}
	return &theme{name: defaultThemeName, files: files}, nil
}

// readBuiltinTheme reads an embedded theme's files.
func readBuiltinTheme(name string) (map[string][]byte, error) {
	sub, err := fs.Sub(builtinThemes, "themes/"+name)
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, themeStylesheet); err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		b, rerr := fs.ReadFile(sub, p)
		if rerr != nil {
			return rerr
		}
		files[p] = b
		return nil
	})
	return files, err
}

// readThemeDir reads a project's theme directory. Only regular files are read: a
// symbolic link is skipped rather than followed, so a theme cannot pull files from
// outside the project into the generated site.
func readThemeDir(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	total := 0
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // a symlink, device, socket, ...
		}
		if len(files) >= maxThemeFiles {
			return fmt.Errorf("more than %d files", maxThemeFiles)
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		total += len(b)
		if total > maxThemeBytes {
			return fmt.Errorf("larger than %s", humanSize(maxThemeBytes))
		}
		rel, rerr := filepath.Rel(dir, p)
		if rerr != nil {
			return rerr
		}
		files[filepath.ToSlash(rel)] = b
		return nil
	})
	return files, err
}

// write puts the theme's files under assets/theme/ in the output directory.
func (t *theme) write(outDir string) error {
	names := make([]string, 0, len(t.files))
	for n := range t.files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := writeAsset(outDir, themeAssetsDir+"/"+n, t.files[n]); err != nil {
			return err
		}
	}
	return nil
}

// writeAsset writes one file under the output directory, creating its folders.
func writeAsset(outDir, rel string, data []byte) error {
	full := filepath.Join(outDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", rel, err)
	}
	return nil
}
