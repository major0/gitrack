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
// user-facing documentation must not cite them.
var adrRef = regexp.MustCompile(`ADR-\d{4}|docs/adr/`)

// checkNoADRRefs fails the test for every ADR citation in the file at path,
// naming the file and line.
func checkNoADRRefs(t *testing.T, path string) error {
	t.Helper()
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
}

// TestUserFacingDocsDoNotCiteADRs checks the repository README and walks docs/,
// skipping the adr/ directory itself (excluded from the published site by
// default). Any ADR citation fails the test.
func TestUserFacingDocsDoNotCiteADRs(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	docs := filepath.Join(repoRoot, "docs")
	if _, err := os.Stat(docs); err != nil {
		t.Skipf("docs tree not found at %s: %v", docs, err)
	}

	if err := checkNoADRRefs(t, filepath.Join(repoRoot, "README.md")); err != nil {
		t.Fatalf("reading README.md: %v", err)
	}

	err := filepath.WalkDir(docs, func(path string, d fs.DirEntry, err error) error {
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
		return checkNoADRRefs(t, path)
	})
	if err != nil {
		t.Fatalf("walking docs: %v", err)
	}
}
