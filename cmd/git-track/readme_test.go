package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// missingFromReadme returns the commands whose name does not appear in the
// README as inline code (`name`), so a command added to the CLI without a README
// mention is caught.
func missingFromReadme(readme string) []string {
	var missing []string
	for _, c := range commands {
		if !strings.Contains(readme, "`"+c.Name+"`") {
			missing = append(missing, c.Name)
		}
	}
	return missing
}

// TestReadmeListsEveryCommand keeps the README's status section honest: the
// command table in this package is the source of truth, and every command in it
// must be named in the README.
func TestReadmeListsEveryCommand(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Skipf("README not found: %v", err)
	}
	if missing := missingFromReadme(string(b)); len(missing) > 0 {
		t.Errorf("README.md does not mention these commands: %v", missing)
	}
}

func TestMissingFromReadmeDetectsGaps(t *testing.T) {
	// a README that names only one command must report all the others
	got := missingFromReadme("Only `init` is documented here.")
	if len(got) != len(commands)-1 {
		t.Fatalf("reported %d missing, want %d", len(got), len(commands)-1)
	}
	for _, name := range got {
		if name == "init" {
			t.Errorf("init is documented but was reported missing")
		}
	}
	if len(missingFromReadme("")) != len(commands) {
		t.Errorf("an empty README must report every command missing")
	}
}
