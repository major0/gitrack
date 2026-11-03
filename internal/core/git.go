package core

import (
	"fmt"
	"os/exec"
	"regexp"
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
	return scanDirectives(message)[strings.ToLower(handle)]
}

// closingKeyword reports whether w (lowercased, trailing ':' already stripped)
// is a closing keyword (GitHub + GitLab union, ADR-0035).
func closingKeyword(w string) bool {
	switch w {
	case "close", "closes", "closed", "closing",
		"fix", "fixes", "fixed", "fixing",
		"resolve", "resolves", "resolved", "resolving",
		"implement", "implements", "implemented", "implementing":
		return true
	}
	return false
}

// relatesKeyword reports whether w is a relates keyword.
func relatesKeyword(w string) bool {
	switch w {
	case "relate", "relates", "related":
		return true
	}
	return false
}

// scanDirectives walks the message left to right and attributes a verb to every
// #<handle> by RUN (ADR-0035): a closing/relates keyword opens a run, and the
// consecutive #refs that follow it — separated only by whitespace, commas, or
// the word "and" (plus an optional "to"/"issue(s)" filler after the keyword) —
// all receive that verb. Any other word ends the run; a ref outside a run is a
// "mention". Returns handle(lowercased) -> strongest verb (closes>relates>mention).
func scanDirectives(message string) map[string]string {
	rank := map[string]int{"mentions": 0, "relates": 1, "closes": 2}
	out := map[string]string{}
	assign := func(handle, verb string) {
		if cur, ok := out[handle]; !ok || rank[verb] > rank[cur] {
			out[handle] = verb
		}
	}
	// Tokenize on whitespace and commas, keeping each token.
	tokens := strings.FieldsFunc(message, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ','
	})
	run := "" // active run verb: "closes", "relates", or "" (none)
	for _, tok := range tokens {
		low := strings.ToLower(tok)
		switch {
		case strings.HasPrefix(tok, "#"):
			handle := strings.ToLower(strings.TrimPrefix(tok, "#"))
			if handle == "" {
				run = ""
				continue
			}
			if run != "" {
				assign(handle, run)
			} else {
				assign(handle, "mentions")
			}
		case closingKeyword(strings.TrimRight(low, ":")):
			run = "closes"
		case relatesKeyword(strings.TrimRight(low, ":")):
			run = "relates"
		case low == "and" || low == "to" || low == "issue" || low == "issues":
			// list/filler connectors keep the run open (ADR-0035)
		default:
			run = "" // any other word ends the run
		}
	}
	return out
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

// StageTracker stages the tracker path into the index (git add .gitrack),
// without committing. Used by the commit wrapper so a resolve's tracker change
// joins the user's already-staged files in one revision.
func (r *Repo) StageTracker() error {
	if out, err := exec.Command("git", "-C", r.Root, "add", "--", Dir).CombinedOutput(); err != nil {
		return fmt.Errorf("git add %s: %w: %s", Dir, err, out)
	}
	return nil
}

// CommitStaged commits the WHOLE staged index with the given message, using the
// system git (user identity, config, hooks). Unlike Commit (which scopes to
// pathspecs), this is the `git track commit` wrapper: it records the user's
// currently-staged files together with any tracker changes already staged, as a
// single revision (ADR-0032).
func (r *Repo) CommitStaged(message string) error {
	if out, err := exec.Command("git", "-C", r.Root, "commit", "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}

// Directive is a ref reference found in a commit message (ADR-0033): a handle
// and the verb that governs it.
type Directive struct {
	Handle string // the #<handle> text (without the leading #)
	Verb   string // "closes" | "relates" | "mentions"
}

// directiveToken matches a #<handle> in a message. Handle charset mirrors the
// reference syntax (ADR-0011): [0-9a-z][0-9a-z-]*.
var directiveToken = regexp.MustCompile(`#([0-9a-z][0-9a-z-]*)`)

// ParseDirectives finds every #<handle> in a commit message and classifies its
// verb (ADR-0033), reusing the mention classifier per occurrence. Duplicate
// handles keep the strongest verb seen (closes > relates > mentions).
func ParseDirectives(message string) []Directive {
	verbs := scanDirectives(message)
	// Emit in first-appearance order of each distinct handle.
	var out []Directive
	seen := map[string]bool{}
	for _, m := range directiveToken.FindAllStringSubmatch(message, -1) {
		h := strings.ToLower(m[1])
		if seen[h] {
			continue
		}
		seen[h] = true
		if v, ok := verbs[h]; ok {
			out = append(out, Directive{Handle: m[1], Verb: v})
		}
	}
	return out
}

// gitOut runs a git command in root and returns its stdout. A thin helper for
// derived read-only queries (author/date, mentions). Errors (e.g. no history)
// are returned so callers can degrade gracefully.
func gitOut(root string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
