package core

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Commit stages and records a commit scoped to the given pathspecs, using the
// system git so the user's own identity, config, and hooks apply (gitrack never
// injects an identity). Only the listed pathspecs are staged and committed
// (`git add -- <paths>` then `git commit -- <paths>`), so changes the user has
// staged OUTSIDE these paths are never swept into the commit and remain staged
// (ADR-0028). Pathspecs are relative to the repo root.
func (r *Repo) Commit(message string, pathspecs ...string) error {
	if len(pathspecs) == 0 {
		return fmt.Errorf("commit requires at least one pathspec")
	}
	addArgs := append([]string{"-C", r.Root, "add", "--"}, pathspecs...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	commitArgs := append([]string{"-C", r.Root, "commit", "-m", message, "--"}, pathspecs...)
	if out, err := exec.Command("git", commitArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}

// Mention is one commit that references a track in its message (ADR-0031).
type Mention struct {
	SHA     string // full commit hash
	Short   string // abbreviated hash
	Subject string // first line of the message
	Verb    string // "closes" | "relates" | "mentions"
}

// handles returns the typeable handles a commit message might use to reference a
// track: its slug, any slug aliases, and a short (12-hex) uuid prefix.
func (r *Repo) handles(id string) ([]string, error) {
	var hs []string
	if slug, _ := r.GetField(id, "slug"); slug != "" {
		hs = append(hs, slug)
	}
	aliases, err := r.listMulti(id, "slug.aliases")
	if err != nil {
		return nil, err
	}
	hs = append(hs, aliases...)
	hs = append(hs, strings.ReplaceAll(id, "-", "")[:12])
	return hs, nil
}

// classifyMention determines the mention verb for a message referencing handle
// (ADR-0031): closes/resolves/fixes -> "closes"; relates/related to ->
// "relates"; otherwise "mentions". Matching is lowercase and tolerant of
// punctuation between the keyword and the ref.
func classifyMention(message, handle string) string {
	msg := strings.ToLower(message)
	ref := "#" + strings.ToLower(handle)
	idx := strings.Index(msg, ref)
	if idx < 0 {
		return ""
	}
	// Look at the words immediately before the ref.
	before := strings.TrimRight(msg[:idx], " \t")
	switch {
	case hasSuffixWord(before, "closes"), hasSuffixWord(before, "close"),
		hasSuffixWord(before, "resolves"), hasSuffixWord(before, "resolve"),
		hasSuffixWord(before, "fixes"), hasSuffixWord(before, "fix"),
		hasSuffixWord(before, "fixed"):
		return "closes"
	case hasSuffixWord(before, "relates to"), hasSuffixWord(before, "related to"),
		hasSuffixWord(before, "relates"), hasSuffixWord(before, "related"):
		return "relates"
	default:
		return "mentions"
	}
}

// hasSuffixWord reports whether s ends with word on a word boundary.
func hasSuffixWord(s, word string) bool {
	if !strings.HasSuffix(s, word) {
		return false
	}
	rest := s[:len(s)-len(word)]
	return rest == "" || strings.HasSuffix(rest, " ") || strings.HasSuffix(rest, "\n") ||
		strings.HasSuffix(rest, "\t")
}

// Mentions returns the commits whose messages reference the track, derived from
// git history with no stored state (ADR-0031). It is portable: a fixed-string
// `git log --grep` candidate filter (no PCRE), then classification in Go.
func (r *Repo) Mentions(id string) ([]Mention, error) {
	handles, err := r.handles(id)
	if err != nil {
		return nil, err
	}
	seen := map[string]Mention{}
	for _, h := range handles {
		// Candidate commits: fixed-string match on "#<handle>" (no PCRE).
		out, err := exec.Command("git", "-C", r.Root, "log",
			"--format=%H", "--grep=#"+h, "-F").Output()
		if err != nil {
			// no commits / not a git repo yet: treat as no mentions
			return mentionSlice(seen), nil
		}
		for _, sha := range strings.Fields(string(out)) {
			if _, ok := seen[sha]; ok {
				continue
			}
			bodyOut, err := exec.Command("git", "-C", r.Root, "log", "-1",
				"--format=%B", sha).Output()
			if err != nil {
				continue
			}
			body := string(bodyOut)
			verb := classifyMention(body, h)
			if verb == "" {
				continue // the fixed-string hit was not an actual #<handle> token boundary
			}
			subj := strings.SplitN(strings.TrimSpace(body), "\n", 2)[0]
			short := sha
			if len(short) > 8 {
				short = short[:8]
			}
			seen[sha] = Mention{SHA: sha, Short: short, Subject: subj, Verb: verb}
		}
	}
	return mentionSlice(seen), nil
}

// mentionSlice flattens the dedup map, newest-first is approximated by SHA set
// order; callers that need strict order can sort by a commit date later.
func mentionSlice(m map[string]Mention) []Mention {
	out := make([]Mention, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SHA < out[j].SHA })
	return out
}
