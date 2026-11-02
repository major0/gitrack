package main

import (
	"strings"
	"testing"
)

func TestManPageFromCommandData(t *testing.T) {
	man := manPage()

	// roff structure
	for _, must := range []string{".TH GIT-TRACK 1", ".SH NAME", ".SH SYNOPSIS",
		".SH COMMANDS", ".SH FILES"} {
		if !strings.Contains(man, must) {
			t.Errorf("man page missing %q", must)
		}
	}
	// single source: every command appears (bold .B <name>) so --help and the
	// man page cannot drift (ADR-0016). Names are roff-escaped, so compare
	// against the escaped form (hyphens become \-).
	for _, c := range commands {
		if !strings.Contains(man, ".B "+roffEscape(c.Name)) {
			t.Errorf("man page missing command %q", c.Name)
		}
	}
	// hyphens in synopses are roff-escaped so flags like -k don't start a request
	if strings.Contains(man, "[-k kind]") {
		t.Error("unescaped hyphen in man output (should be \\- escaped)")
	}
}

func TestUsageTextFromCommandData(t *testing.T) {
	u := usageText()
	for _, c := range commands {
		if !strings.Contains(u, c.Name) {
			t.Errorf("usage missing command %q", c.Name)
		}
	}
	if !strings.Contains(u, tagline) {
		t.Error("usage missing tagline")
	}
}
