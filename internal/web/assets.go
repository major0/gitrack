package web

import (
	"fmt"
	"sort"
)

// Asset paths, relative to the site root. Every stylesheet and script a page uses
// is a file here; pages carry no inline CSS or script (ADR-0048).
const (
	highlightAsset = assetsDir + "/highlight.css"
	copyAsset      = assetsDir + "/gitrack.js"
	boardAsset     = assetsDir + "/board.js"
	themeCSSAsset  = themeAssetsDir + "/" + themeStylesheet
	themeJSAsset   = themeAssetsDir + "/" + themeScript
)

// boardScript draws the dependency graph with Mermaid when the page is online.
// It is a classic script that loads the library with a dynamic import(), because
// a module script is refused when a page is opened from file://. Offline the
// import fails, the graph source stays readable in its <pre>, and the Ready and
// Blocked lists are the complete view (ADR-0029).
const boardScript = `(function () {
  var nodes = document.querySelectorAll('pre.mermaid');
  if (!nodes.length) { return; }
  import('https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs').then(function (m) {
    m.default.initialize({ startOnLoad: false });
    return m.default.run({ nodes: nodes });
  }).catch(function () {});
})();`

// assetUse records which optional assets the pages written so far refer to, so
// generation writes exactly those files.
type assetUse struct {
	highlight bool
	copy      bool
	board     bool
}

// scriptsFor lists the scripts a page loads, in order, and notes the optional
// assets it uses. The theme's own script, when it has one, comes first.
func (sc *siteContext) scriptsFor(x pageExtra) []string {
	var out []string
	if sc.theme.hasScript() {
		out = append(out, themeJSAsset)
	}
	if x.Code && sc.copyEnabled() {
		sc.used.copy = true
		out = append(out, copyAsset)
	}
	if x.Board {
		sc.used.board = true
		out = append(out, boardAsset)
	}
	return out
}

// writeAssets writes the theme and every optional asset the site used. It
// overwrites only these known files and never clears the directory: the output
// directory may be a repository root.
func (sc *siteContext) writeAssets() error {
	if err := sc.theme.write(sc.outDir); err != nil {
		return err
	}
	files := map[string]string{}
	if sc.used.highlight {
		hl, err := sc.highlighter()
		if err != nil {
			return err
		}
		if hl != nil {
			files[highlightAsset] = string(hl.css)
		}
	}
	if sc.used.copy {
		files[copyAsset] = copyScript
	}
	if sc.used.board {
		files[boardAsset] = boardScript
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := writeAsset(sc.outDir, n, []byte(files[n])); err != nil {
			return fmt.Errorf("writing asset %s: %w", n, err)
		}
	}
	return nil
}
