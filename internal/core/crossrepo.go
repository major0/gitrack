package core

import "strings"

// CrossRepoRef is a parsed cross-repo reference (ADR-0011/0034): a clonable git
// URL and the target track's uuid (possibly a short prefix).
type CrossRepoRef struct {
	RepoURL string
	UUID    string
}

// ParseCrossRepo parses the cross-repo reference form "<repo-url>?track=<uuid>"
// (the leading '#' is optional) into its parts. It is a pure, offline parse: it
// never contacts the remote (ADR-0034). ok is false when s is not a cross-repo
// reference (no "?track=" marker, empty url, or empty uuid).
func ParseCrossRepo(s string) (ref CrossRepoRef, ok bool) {
	s = strings.TrimPrefix(s, "#")
	url, uuid, found := strings.Cut(s, "?track=")
	if !found || url == "" || uuid == "" {
		return CrossRepoRef{}, false
	}
	// uuid part is a bare handle: hex/uuid only (no further query params here).
	if i := strings.IndexAny(uuid, "&#"); i >= 0 {
		uuid = uuid[:i]
	}
	if uuid == "" {
		return CrossRepoRef{}, false
	}
	return CrossRepoRef{RepoURL: url, UUID: uuid}, true
}

// RepoLabel returns a short human label for a repo URL — its last path segment
// without a trailing ".git" — for display in a cross-repo link.
func (c CrossRepoRef) RepoLabel() string {
	u := strings.TrimSuffix(c.RepoURL, ".git")
	u = strings.TrimRight(u, "/")
	if i := strings.LastIndexAny(u, "/:"); i >= 0 {
		u = u[i+1:]
	}
	if u == "" {
		return c.RepoURL
	}
	return u
}

// ShortUUID returns the uuid truncated to 8 chars for display (or the whole
// value if shorter).
func (c CrossRepoRef) ShortUUID() string {
	hex := strings.ReplaceAll(c.UUID, "-", "")
	if len(hex) >= 8 {
		return hex[:8]
	}
	return c.UUID
}
