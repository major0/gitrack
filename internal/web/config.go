package web

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// siteConfigDir is the tracker-relative directory holding site generator
// config (ADR-0036/0037): .gitrack/config/site/.
const siteConfigDir = "config/site"

// defaultSections is the section set used when no sections config exists: the
// issues tracker only, preserving pre-ADR-0037 behavior (ADR-0036 §3).
var defaultSections = []string{sectionIssues}

// siteConfig is the resolved site generator configuration. It is deliberately
// small: the enabled section names in order, and the theme name. Absent config
// yields the defaults (issues only, built-in theme), so a repo with no
// config/site/ generates exactly today's output (ADR-0037 verification).
type siteConfig struct {
	// Sections is the ordered list of enabled section names.
	Sections []string
	// Theme is the theme name; "" selects the built-in default theme.
	Theme string
}

// loadSiteConfig reads .gitrack/config/site/ for the repo. A missing directory
// or missing file falls back to defaults. Config is a guardrail, not
// enforcement (ADR-0026): unknown section names are dropped by the section
// registry at build time, not rejected here.
func loadSiteConfig(repo *core.Repo) (siteConfig, error) {
	base := filepath.Join(repo.Root, core.Dir, siteConfigDir)
	sections, err := readConfigLines(filepath.Join(base, "sections"))
	if err != nil {
		return siteConfig{}, err
	}
	if len(sections) == 0 {
		sections = defaultSections
	}
	theme, err := readConfigLines(filepath.Join(base, "theme"))
	if err != nil {
		return siteConfig{}, err
	}
	cfg := siteConfig{Sections: sections}
	if len(theme) > 0 {
		cfg.Theme = theme[0]
	}
	return cfg, nil
}

// readConfigFile returns the raw contents of a config file; a missing file
// yields "" with no error. Used where structure (e.g. the docs layout's
// indentation) must be preserved rather than line-stripped.
func readConfigFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(b), nil
}

// readConfigValue returns the first non-empty, non-comment line of a
// single-value config file, or def when the file is missing or has no such line.
func readConfigValue(path, def string) (string, error) {
	raw, err := readConfigFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line, nil
	}
	return def, nil
}

// readConfigLines returns the non-empty, comment-stripped lines of a config
// file; a missing file yields nil with no error. This mirrors core's internal
// readLines convention (ADR-0026) for the web-owned site config.
func readConfigLines(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}
