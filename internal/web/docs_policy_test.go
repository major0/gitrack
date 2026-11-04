package web

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// adrRef matches an ADR citation ("ADR-0019") or a link into the ADR directory.
// ADRs are contributor design records, not shipped to users of the tooling, so
// the published documentation must not cite them.
var adrRef = regexp.MustCompile(`ADR-\d{4}|docs/adr/`)

// TestUserFacingDocsDoNotCiteADRs walks the repository's docs/ tree, skipping the
// adr/ directory itself (which is excluded from the published site by default),
// and fails on any ADR citation, naming the file and line.
func TestUserFacingDocsDoNotCiteADRs(t *testing.T) {
	root := filepath.Join("..", "..", "docs")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("docs tree not found at %s: %v", root, err)
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "adr" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		sc := bufio.NewScanner(f)
		for n := 1; sc.Scan(); n++ {
			if loc := adrRef.FindString(sc.Text()); loc != "" {
				t.Errorf("%s:%d cites %q; user-facing docs must not reference ADRs", path, n, loc)
			}
		}
		return sc.Err()
	})
	if err != nil {
		t.Fatalf("walking docs: %v", err)
	}
}
