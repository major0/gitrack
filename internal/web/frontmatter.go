package web

import "strings"

// frontMatter is the minimal metadata parsed from a page/post's leading
// `---`-delimited block (ADR-0039). Only the keys the portal/blog sections use
// are recognized; unknown keys are kept in Extra for forward tolerance but are
// not otherwise interpreted.
type frontMatter struct {
	Title string
	Date  string // YYYY-MM-DD, blog only; "" when absent
	Extra map[string]string
}

// splitFrontMatter separates an optional leading `---` front-matter block from
// the Markdown body. A file that does not begin with a `---` line has no front
// matter: the whole content is the body. The parser is a tiny key:value reader
// (no YAML dependency, ADR-0038/0039): each line is split on the first ":",
// key and value trimmed; blank and comment (#) lines inside the block are
// ignored.
func splitFrontMatter(content string) (frontMatter, string) {
	fm := frontMatter{Extra: map[string]string{}}
	// Must start with a line that is exactly "---".
	rest := content
	// Normalize a leading BOM/whitespace-free check: the first line must be ---.
	first, after, found := cutLine(rest)
	if !found || strings.TrimRight(first, "\r") != "---" {
		return fm, content
	}
	body := after
	for {
		line, next, ok := cutLine(body)
		if !ok {
			// No closing fence: treat the whole thing as body (no front matter).
			return frontMatter{Extra: map[string]string{}}, content
		}
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "---" {
			// End of front matter; the rest is the body.
			applyKnownKeys(&fm)
			return fm, next
		}
		t := strings.TrimSpace(trimmed)
		if t == "" || strings.HasPrefix(t, "#") {
			body = next
			continue
		}
		if i := strings.Index(t, ":"); i >= 0 {
			key := strings.TrimSpace(t[:i])
			val := strings.TrimSpace(t[i+1:])
			if key != "" {
				fm.Extra[strings.ToLower(key)] = val
			}
		}
		body = next
	}
}

// applyKnownKeys promotes recognized Extra keys onto typed fields.
func applyKnownKeys(fm *frontMatter) {
	if v, ok := fm.Extra["title"]; ok {
		fm.Title = v
	}
	if v, ok := fm.Extra["date"]; ok {
		fm.Date = v
	}
}

// cutLine splits off the first line (without its trailing newline) and returns
// the remainder after the newline. found is false when s is empty.
func cutLine(s string) (line, rest string, found bool) {
	if s == "" {
		return "", "", false
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", true
}
