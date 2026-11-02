// Command git-track is the gitrack CLI, installed on PATH so that git's native
// subcommand dispatch runs it as "git track" (ADR-0015/0016). Commands:
// init, create, update, get, set, show, list, search, close, reopen, fsck.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/major0/gitrack/internal/cache"
	"github.com/major0/gitrack/internal/core"
	"github.com/major0/gitrack/internal/server"
	"github.com/major0/gitrack/internal/web"
)

// tagline is the one-line description used in both --help and the man page.
const tagline = "distributed tracker whose state merges with your code"

// command describes one CLI verb. This slice is the single source of truth for
// both the --help text and the generated man page (ADR-0016: one source, no
// drift).
type command struct {
	Name    string // verb
	Args    string // argument synopsis
	Summary string // one-line description
}

var commands = []command{
	{"init", "[--no-commit]", "initialize a tracker (commits the scaffold unless --no-commit)"},
	{"config", "status ...", "edit status workflows (see 'config status')"},
	{"create", "[-k kind] <title>", "create a track, print its id"},
	{"update", "<ref> [flags]", "change a track's title/status/kind/description"},
	{"get", "[--json] <ref>[.path]", "read a field, list, or object (e.g. <ref>.comments.<id>)"},
	{"set", "<ref>.<key> <value>", "write a free-form field (not state/status)"},
	{"show", "<ref>", "display a track (ref: id, short id, or slug)"},
	{"list", "[--state s] [-s status] [-k kind]", "list tracks"},
	{"search", "<query>", "find tracks by text in title/description/slug"},
	{"close", "<ref> [--status=s]", "close a track: state=closed, status=s (default done)"},
	{"reopen", "<ref>", "reopen a track: state=open, status=reopened"},
	{"comment", "<ref> [-m msg] [--reply-to=<cid>]", "add a Markdown comment"},
	{"depends-on", "<ref> <target>", "mark <ref> as blocked by <target>"},
	{"child-of", "<ref> <parent>", "place <ref> under <parent> (one parent)"},
	{"related", "<ref> <target>", "soft link, no scheduling effect"},
	{"duplicate-of", "<ref> <target>", "mark <ref> a duplicate of <target>"},
	{"unlink", "<ref> <rel> <target>", "remove a relationship edge"},
	{"board", "[--graph|--mermaid|--dot]", "ready/blocked work, or the dependency digraph"},
	{"web", "--out <dir>", "generate a self-contained static site (offline-capable)"},
	{"serve", "[--addr 127.0.0.1:0]", "live loopback preview server (random port)"},
	{"fsck", "[--write]", "verify (or rebuild) the derived cache"},
}

// overview is the shared prose shown after the command list.
const overview = `Every command is a verb. A track has two axes: a fixed state (open|closed)
changed only by create/close/reopen, and a customizable status workflow
(built-ins new/reopened/done plus user-defined statuses in config/statuses/)
moved by 'update -s'. update flags: -t title | -s status | -k kind |
-d description.

Every mutating command commits its change as its own revision scoped to
.gitrack/ (unrelated staged changes are never captured). --include <path>
(repeatable) bundles a related file into that revision; --no-commit leaves the
change staged instead. Commits use your git identity.

See docs/FORMAT.md for the on-disk format.`

// usageText renders the --help screen from the command data.
func usageText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "git-track - %s\n\nusage: git track <command> [args]\n\ncommands:\n", tagline)
	for _, c := range commands {
		synopsis := c.Name
		if c.Args != "" {
			synopsis += " " + c.Args
		}
		fmt.Fprintf(&b, "  %-42s%s\n", synopsis, c.Summary)
	}
	b.WriteString("\n")
	b.WriteString(overview)
	b.WriteString("\n")
	return b.String()
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "git-track:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(usageText())
		return nil
	}
	if args[0] == "help" {
		// `git track help --man` emits the roff man page from the same command
		// data (ADR-0016 single source). Plain `help` prints usage.
		if len(args) > 1 && args[1] == "--man" {
			fmt.Print(manPage())
			return nil
		}
		fmt.Print(usageText())
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "init":
		return cmdInit(rest)
	case "config":
		return cmdConfig(rest)
	case "create":
		return cmdCreate(rest)
	case "update":
		return cmdUpdate(rest)
	case "get":
		return cmdGet(rest)
	case "set":
		return cmdSet(rest)
	case "show":
		return cmdShow(rest)
	case "list", "ls":
		return cmdList(rest)
	case "search":
		return cmdSearch(rest)
	case "close":
		return cmdClose(rest)
	case "reopen":
		return cmdReopen(rest)
	case "comment":
		return cmdComment(rest)
	case "depends-on", "child-of", "related", "duplicate-of":
		return cmdLink(cmd, rest)
	case "unlink":
		return cmdUnlink(rest)
	case "board":
		return cmdBoard(rest)
	case "web":
		return cmdWeb(rest)
	case "serve":
		return cmdServe(rest)
	case "fsck":
		return cmdFsck(rest)
	default:
		return fmt.Errorf("unknown command %q (try 'git track --help')", cmd)
	}
}

// rebuildCache regenerates the derived cache after a mutation so index reads
// stay correct through the tool (ADR-0014).
func rebuildCache(repo *core.Repo) error {
	if _, err := cache.Rebuild(repo); err != nil {
		return fmt.Errorf("updating cache: %w", err)
	}
	return nil
}

func openRepo() (*core.Repo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return core.Open(wd)
}

func cmdInit(args []string) error {
	commit := true
	for _, a := range args {
		switch a {
		case "--no-commit":
			commit = false
		default:
			return fmt.Errorf("unknown init option %q", a)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	repo, err := core.Init(wd)
	if err != nil {
		return err
	}
	fmt.Printf("initialized gitrack tracker in %s/%s\n", repo.Root, core.Dir)
	// init commits the scaffold by default (a natural one-time bootstrap
	// commit); --no-commit leaves it staged for the user. Other mutating
	// commands never auto-commit, so tracker changes ride along with the
	// related code change.
	if commit {
		if err := repo.Commit("gitrack: initialize tracker", core.Dir); err != nil {
			return fmt.Errorf("committing scaffold (use --no-commit to skip): %w", err)
		}
		fmt.Println("committed the tracker scaffold")
	}
	return nil
}

// commitOpts holds the commit-related flags shared by mutating commands
// (ADR-0028): --no-commit opts out, --include adds extra pathspecs.
type commitOpts struct {
	noCommit bool
	include  []string
}

// extractCommitFlags pulls --no-commit and repeated --include <path> out of
// args, returning the remaining args and the parsed commit options.
func extractCommitFlags(args []string) ([]string, commitOpts, error) {
	var rest []string
	var opts commitOpts
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--no-commit":
			opts.noCommit = true
		case "--include":
			if i+1 >= len(args) {
				return nil, opts, fmt.Errorf("--include requires a path")
			}
			i++
			opts.include = append(opts.include, args[i])
		default:
			rest = append(rest, args[i])
		}
	}
	return rest, opts, nil
}

// commitChange commits the tracker change as its own scoped revision unless
// --no-commit was given (ADR-0028). The scope is .gitrack/ plus any --include
// paths; unrelated staged changes are never captured.
func commitChange(repo *core.Repo, opts commitOpts, subject string) error {
	if opts.noCommit {
		return nil
	}
	paths := append([]string{core.Dir}, opts.include...)
	if err := repo.Commit(subject, paths...); err != nil {
		return fmt.Errorf("committing change (use --no-commit to skip): %w", err)
	}
	return nil
}

const configStatusUsage = `usage: git track config status <verb>
  list                        list statuses and their allowed next-sets
  add <name> [--next a,b,c]   add a user-defined status
  set-next <name> [next...]   replace a status's allowed next-set (empty = any)
  rm <name>                   remove a user-defined status (built-ins refused)`

func cmdConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: git track config <domain> <verb>; domains: status")
	}
	switch args[0] {
	case "status":
		return cmdConfigStatus(args[1:])
	default:
		return fmt.Errorf("unknown config domain %q (have: status)", args[0])
	}
}

func cmdConfigStatus(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("%s", configStatusUsage)
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return configStatusList(repo) // read-only, no commit
	case "add":
		if err := configStatusAdd(repo, rest); err != nil {
			return err
		}
		return commitChange(repo, cOpts, "gitrack: config status add")
	case "set-next":
		if len(rest) == 0 {
			return fmt.Errorf("set-next requires a status name")
		}
		if err := repo.SetNextStatuses(rest[0], rest[1:]); err != nil {
			return err
		}
		return commitChange(repo, cOpts, "gitrack: config status set-next "+rest[0])
	case "rm":
		if len(rest) != 1 {
			return fmt.Errorf("rm requires exactly one status name")
		}
		if err := repo.RemoveStatus(rest[0]); err != nil {
			return err
		}
		return commitChange(repo, cOpts, "gitrack: config status rm "+rest[0])
	default:
		return fmt.Errorf("unknown config status verb %q\n%s", verb, configStatusUsage)
	}
}

func configStatusList(repo *core.Repo) error {
	statuses, err := repo.Statuses()
	if err != nil {
		return err
	}
	for _, s := range statuses {
		next, err := repo.NextStatuses(s)
		if err != nil {
			return err
		}
		tag := ""
		if core.IsBuiltinStatus(s) {
			tag = " (built-in)"
		}
		nextStr := "any"
		if len(next) > 0 {
			nextStr = strings.Join(next, ", ")
		}
		fmt.Printf("%-14s%s -> %s\n", s, tag, nextStr)
	}
	return nil
}

func configStatusAdd(repo *core.Repo, args []string) error {
	var name string
	var next []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--next":
			if i+1 >= len(args) {
				return fmt.Errorf("--next requires a comma-separated list")
			}
			i++
			for _, part := range strings.Split(args[i], ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					next = append(next, part)
				}
			}
		default:
			if name != "" {
				return fmt.Errorf("add takes one status name, got extra %q", args[i])
			}
			name = args[i]
		}
	}
	if name == "" {
		return fmt.Errorf("add requires a status name")
	}
	if err := repo.AddStatus(name, next); err != nil {
		return err
	}
	fmt.Printf("added status %q\n", name)
	return nil
}

func cmdCreate(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	var kind string
	var title []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-k", "--kind":
			if i+1 >= len(args) {
				return fmt.Errorf("-k requires a kind")
			}
			i++
			kind = args[i]
		default:
			title = append(title, args[i])
		}
	}
	if len(title) == 0 {
		return fmt.Errorf("create requires a title")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Create(core.NewTrackOpts{Kind: kind, Title: strings.Join(title, " ")})
	if err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	slug, _ := repo.GetField(id, "slug")
	if err := commitChange(repo, cOpts, "gitrack: create "+slug); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func cmdShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("show requires exactly one track reference")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(args[0])
	if err != nil {
		return err
	}
	t, err := repo.Load(id)
	if err != nil {
		return err
	}
	fmt.Printf("id:      %s\n", t.ID)
	fmt.Printf("slug:    %s\n", t.Slug)
	fmt.Printf("kind:    %s\n", t.Kind)
	fmt.Printf("title:   %s\n", t.Title)
	fmt.Printf("state:   %s\n", t.State)
	fmt.Printf("status:  %s\n", t.Status)
	for _, rel := range core.RelationKinds {
		if targets := t.Relations[rel]; len(targets) > 0 {
			fmt.Printf("%s: %s\n", rel, strings.Join(targets, ", "))
		}
	}
	// Commit mentions are derived from git history (ADR-0031).
	if mentions, err := repo.Mentions(id); err == nil && len(mentions) > 0 {
		fmt.Println("mentions:")
		for _, m := range mentions {
			fmt.Printf("  %s %-8s %s\n", m.Short, m.Verb, m.Subject)
		}
	}
	// Blocked/unblocked is derived from relationships + state (ADR-0005).
	if blockers, err := repo.Blockers(id); err == nil && len(blockers) > 0 {
		unblocked, _ := repo.IsUnblocked(id)
		label := "blocked"
		if unblocked {
			label = "unblocked"
		}
		short := make([]string, len(blockers))
		for i, b := range blockers {
			short[i] = b[:8]
		}
		fmt.Printf("%s by: %s\n", label, strings.Join(short, ", "))
	}
	if t.Description != "" {
		fmt.Printf("\n%s\n", t.Description)
	}
	return nil
}

func cmdList(args []string) error {
	var wantState, wantStatus, wantKind string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state":
			if i+1 >= len(args) {
				return fmt.Errorf("--state requires a value")
			}
			i++
			wantState = args[i]
		case "-s", "--status":
			if i+1 >= len(args) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			wantStatus = args[i]
		case "-k", "--kind":
			if i+1 >= len(args) {
				return fmt.Errorf("-k requires a kind")
			}
			i++
			wantKind = args[i]
		default:
			return fmt.Errorf("unknown list option %q", args[i])
		}
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	ids, err := repo.ListIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		t, err := repo.Load(id)
		if err != nil {
			return err
		}
		if wantState != "" && t.State != wantState {
			continue
		}
		if wantStatus != "" && t.Status != wantStatus {
			continue
		}
		if wantKind != "" && t.Kind != wantKind {
			continue
		}
		short := strings.ReplaceAll(id, "-", "")[:8]
		fmt.Printf("%s  %-6s %-11s %-8s %s\n", short, t.State, t.Status, t.Kind, t.Title)
	}
	return nil
}

func cmdUpdate(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("update requires a track reference")
	}
	ref := args[0]
	rest := args[1:]
	fields := map[string]string{} // field -> value, applied after resolve
	var newStatus string
	for i := 0; i < len(rest); i++ {
		if i+1 >= len(rest) {
			return fmt.Errorf("%s requires a value", rest[i])
		}
		val := rest[i+1]
		switch rest[i] {
		case "-t", "--title":
			fields["title"] = val
		case "-k", "--kind":
			fields["kind"] = val
		case "-d", "--description":
			fields["description"] = val
		case "-s", "--status":
			// status moves through the workflow (ADR-0019); state (open/closed)
			// changes only via close/reopen.
			newStatus = val
		default:
			return fmt.Errorf("unknown update flag %q", rest[i])
		}
		i++
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(ref)
	if err != nil {
		return err
	}
	for name, val := range fields {
		if err := repo.SetField(id, name, val); err != nil {
			return err
		}
	}
	if newStatus != "" {
		if err := repo.SetStatus(id, newStatus); err != nil {
			return err
		}
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	slug, _ := repo.GetField(id, "slug")
	if err := commitChange(repo, cOpts, "gitrack: update "+slug); err != nil {
		return err
	}
	fmt.Printf("updated %s\n", id)
	return nil
}

func cmdGet(args []string) error {
	asJSON := false
	var accessor string
	for _, a := range args {
		switch a {
		case "--json":
			asJSON = true
		default:
			if accessor != "" {
				return fmt.Errorf("get takes one accessor, got extra %q", a)
			}
			accessor = a
		}
	}
	if accessor == "" {
		return fmt.Errorf("get requires <ref>[.path]")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	v, err := repo.Get(accessor)
	if err != nil {
		return err
	}
	if asJSON {
		return printJSON(v)
	}
	return printValue(v)
}

// printValue renders a resolved Value in human-readable form.
func printValue(v *core.Value) error {
	switch v.Kind {
	case "scalar":
		fmt.Println(v.Scalar)
	case "list":
		for _, e := range v.List {
			fmt.Println(e)
		}
	case "object":
		t := v.Object
		fmt.Printf("id:      %s\nslug:    %s\nkind:    %s\ntitle:   %s\nstate:   %s\nstatus:  %s\n",
			t.ID, t.Slug, t.Kind, t.Title, t.State, t.Status)
	}
	return nil
}

// printJSON renders a resolved Value as JSON for scripting.
func printJSON(v *core.Value) error {
	var payload any
	switch v.Kind {
	case "scalar":
		payload = v.Scalar
	case "list":
		payload = v.List
	case "object":
		payload = v.Object
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

func cmdSet(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) != 2 {
		return fmt.Errorf("set requires <ref>.<key> <value>")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	if err := repo.Set(args[0], args[1]); err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	if err := commitChange(repo, cOpts, "gitrack: set "+args[0]); err != nil {
		return err
	}
	fmt.Printf("set %s\n", args[0])
	return nil
}

func cmdSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search requires a query")
	}
	query := strings.ToLower(strings.Join(args, " "))
	repo, err := openRepo()
	if err != nil {
		return err
	}
	ids, err := repo.ListIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		t, err := repo.Load(id)
		if err != nil {
			return err
		}
		hay := strings.ToLower(t.Title + "\n" + t.Description + "\n" + t.Slug)
		if strings.Contains(hay, query) {
			short := strings.ReplaceAll(id, "-", "")[:8]
			fmt.Printf("%s  %-6s %-11s %-8s %s\n", short, t.State, t.Status, t.Kind, t.Title)
		}
	}
	return nil
}

func cmdClose(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("close requires a track reference")
	}
	ref := args[0]
	var status string
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		// support --status=value as well as --status value / -s value
		if v, ok := strings.CutPrefix(arg, "--status="); ok {
			status = v
			continue
		}
		switch arg {
		case "-s", "--status":
			if i+1 >= len(rest) {
				return fmt.Errorf("--status requires a value")
			}
			i++
			status = rest[i]
		default:
			return fmt.Errorf("unknown close option %q", arg)
		}
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(ref)
	if err != nil {
		return err
	}
	if err := repo.Close(id, status); err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	slug, _ := repo.GetField(id, "slug")
	if err := commitChange(repo, cOpts, "gitrack: close "+slug); err != nil {
		return err
	}
	fmt.Printf("closed %s\n", id)
	return nil
}

func cmdReopen(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("reopen requires exactly one track reference")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(args[0])
	if err != nil {
		return err
	}
	if err := repo.Reopen(id); err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	slug, _ := repo.GetField(id, "slug")
	if err := commitChange(repo, cOpts, "gitrack: reopen "+slug); err != nil {
		return err
	}
	fmt.Printf("reopened %s\n", id)
	return nil
}

func cmdComment(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("comment requires a track reference")
	}
	ref := args[0]
	var message, replyTo string
	haveMessage := false
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		if v, ok := strings.CutPrefix(arg, "--reply-to="); ok {
			replyTo = v
			continue
		}
		switch arg {
		case "-m", "--message":
			if i+1 >= len(rest) {
				return fmt.Errorf("-m requires a value")
			}
			i++
			message = rest[i]
			haveMessage = true
		case "--reply-to":
			if i+1 >= len(rest) {
				return fmt.Errorf("--reply-to requires a comment id")
			}
			i++
			replyTo = rest[i]
		default:
			return fmt.Errorf("unknown comment option %q", arg)
		}
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(ref)
	if err != nil {
		return err
	}
	// Resolve the reply-to parent (accepts a bare comment uuid).
	if replyTo != "" {
		if _, err := repo.LoadComment(id, replyTo); err != nil {
			return fmt.Errorf("--reply-to: %w", err)
		}
	}

	body := message
	if !haveMessage {
		// Seed the editor: when replying, cite-quote the parent (git-commit style).
		var seed string
		if replyTo != "" {
			parent, err := repo.LoadComment(id, replyTo)
			if err != nil {
				return err
			}
			seed = core.CiteQuote(parent.Body) + "\n"
		}
		body, err = editInEditor(seed)
		if err != nil {
			return err
		}
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("empty comment, aborting")
	}
	cid, err := repo.AddComment(id, body, replyTo)
	if err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	if err := commitChange(repo, cOpts, "gitrack: comment on "+id[:8]); err != nil {
		return err
	}
	fmt.Printf("added comment %s\n", cid)
	return nil
}

// editInEditor opens $EDITOR (falling back to $VISUAL, then vi) on a temp file
// seeded with the given content and returns the edited text. Lines beginning
// with '#' are stripped, git-commit style, so seeded cite-quotes ('>') survive
// but instruction comments do not.
func editInEditor(seed string) (string, error) {
	editor := firstNonEmpty(os.Getenv("EDITOR"), os.Getenv("VISUAL"), "vi")
	f, err := os.CreateTemp("", "gitrack-comment-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	instructions := "# Enter your comment above. Lines starting with '#' are ignored.\n" +
		"# A blockquote ('>') citing the parent is kept.\n"
	if _, err := f.WriteString(seed + instructions); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	cmd := exec.Command("sh", "-c", editor+" \""+path+"\"")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running editor %q: %w", editor, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}
	return stripCommentLines(string(raw)), nil
}

// stripCommentLines removes lines beginning with '#' (after optional leading
// whitespace), git-commit style.
func stripCommentLines(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// firstNonEmpty returns the first non-empty string among its arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// cmdLink handles the relation-named verbs (depends-on/child-of/related/
// duplicate-of), all routed through core.Link. rel is the command name.
func cmdLink(rel string, args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) != 2 {
		return fmt.Errorf("%s requires <ref> <target>", rel)
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(args[0])
	if err != nil {
		return err
	}
	target, err := repo.Resolve(args[1])
	if err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if err := repo.Link(id, rel, target); err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	if err := commitChange(repo, cOpts, fmt.Sprintf("gitrack: %s %s -> %s", rel, id[:8], target[:8])); err != nil {
		return err
	}
	fmt.Printf("%s: %s -> %s\n", rel, id[:8], target[:8])
	return nil
}

func cmdUnlink(args []string) error {
	args, cOpts, err := extractCommitFlags(args)
	if err != nil {
		return err
	}
	if len(args) != 3 {
		return fmt.Errorf("unlink requires <ref> <rel> <target>")
	}
	rel := args[1]
	if !core.IsRelationKind(rel) {
		return fmt.Errorf("unknown relationship %q", rel)
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Resolve(args[0])
	if err != nil {
		return err
	}
	target, err := repo.Resolve(args[2])
	if err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if err := repo.Unlink(id, rel, target); err != nil {
		return err
	}
	if err := rebuildCache(repo); err != nil {
		return err
	}
	if err := commitChange(repo, cOpts, fmt.Sprintf("gitrack: unlink %s %s -> %s", rel, id[:8], target[:8])); err != nil {
		return err
	}
	fmt.Printf("unlinked %s: %s -> %s\n", rel, id[:8], target[:8])
	return nil
}

func cmdBoard(args []string) error {
	mode := "text"
	for _, a := range args {
		switch a {
		case "--graph", "--mermaid":
			mode = "mermaid"
		case "--dot":
			mode = "dot"
		default:
			return fmt.Errorf("unknown board option %q", a)
		}
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	board, err := repo.ComputeBoard()
	if err != nil {
		return err
	}
	switch mode {
	case "mermaid":
		// wrap the shared core renderer in a Markdown fence for terminal/paste use
		fmt.Printf("```mermaid\n%s```\n", board.Mermaid())
	case "dot":
		fmt.Print(board.DOT())
	default:
		printBoardText(board)
	}
	return nil
}

// printBoardText lists open tracks as Ready vs Blocked (ADR-0027). Closed tracks
// are omitted; a blocked track shows its open blockers.
func printBoardText(board *core.Board) {
	byID := map[string]core.BoardEntry{}
	for _, e := range board.Entries {
		byID[e.ID] = e
	}
	var ready, blocked []core.BoardEntry
	for _, e := range board.Entries {
		if !e.Open {
			continue
		}
		if e.Ready() {
			ready = append(ready, e)
		} else {
			blocked = append(blocked, e)
		}
	}
	fmt.Printf("Ready (%d):\n", len(ready))
	for _, e := range ready {
		fmt.Printf("  %s  %-11s %s\n", e.ID[:8], e.Status, e.Title)
	}
	fmt.Printf("\nBlocked (%d):\n", len(blocked))
	for _, e := range blocked {
		labels := make([]string, len(e.OpenBlockers))
		for i, b := range e.OpenBlockers {
			labels[i] = blockerLabel(byID, b)
		}
		fmt.Printf("  %s  %-11s %s  (waiting on: %s)\n",
			e.ID[:8], e.Status, e.Title, strings.Join(labels, ", "))
	}
}

// blockerLabel renders a blocker as "shortid title" when known.
func blockerLabel(byID map[string]core.BoardEntry, id string) string {
	if e, ok := byID[id]; ok {
		return id[:8] + " " + e.Title
	}
	return id[:8]
}

func cmdWeb(args []string) error {
	out := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 >= len(args) {
				return fmt.Errorf("--out requires a directory")
			}
			i++
			out = args[i]
		default:
			return fmt.Errorf("unknown web option %q", args[i])
		}
	}
	if out == "" {
		return fmt.Errorf("web requires --out <dir>")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	n, err := web.Generate(repo, out)
	if err != nil {
		return err
	}
	fmt.Printf("generated static site in %s (%d track pages)\n", out, n)
	return nil
}

func cmdServe(args []string) error {
	addr := "127.0.0.1:0"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 >= len(args) {
				return fmt.Errorf("--addr requires a value")
			}
			i++
			addr = args[i]
		default:
			return fmt.Errorf("unknown serve option %q", args[i])
		}
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	// Clean shutdown on SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(repo, addr, func(url string) {
			fmt.Printf("gitrack serving at %s (Ctrl-C to stop)\n", url)
		})
	}()
	select {
	case <-ctx.Done():
		fmt.Println("\nshutting down")
		return nil
	case err := <-errCh:
		return err
	}
}

func cmdFsck(args []string) error {
	write := false
	for _, a := range args {
		if a == "--write" {
			write = true
		} else {
			return fmt.Errorf("unknown fsck option %q", a)
		}
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	if write {
		rep, err := cache.Rebuild(repo)
		if err != nil {
			return err
		}
		fmt.Println("cache rebuilt from authoritative data")
		// A rebuild fixes the cache, but dangling links and guardrail warnings
		// are authored-data facts a rebuild must not silently fix (ADR-0026).
		reportIssues(rep)
		if len(rep.Dangling) > 0 || len(rep.Warnings) > 0 {
			return errDrift
		}
		return nil
	}
	rep, err := cache.Verify(repo)
	if err != nil {
		return err
	}
	if rep.Clean() {
		fmt.Println("cache is clean; no guardrail divergences")
		return nil
	}
	for _, m := range rep.Missing {
		fmt.Printf("missing:  %s\n", m)
	}
	for _, e := range rep.Extra {
		fmt.Printf("extra:    %s\n", e)
	}
	reportIssues(rep)
	if len(rep.Missing) > 0 || len(rep.Extra) > 0 {
		fmt.Fprintln(os.Stderr, "\nrun 'git track fsck --write' to rebuild the cache")
	}
	// Exit nonzero on any divergence so CI and scripts can gate, matching
	// 'git fsck' and linter conventions.
	return errDrift
}

// errDrift signals that fsck found the cache out of date or a guardrail
// divergence (ADR-0026).
var errDrift = fmt.Errorf("fsck found divergences")

// reportIssues prints dangling links and guardrail warnings (ADR-0026). The
// warnings are advisory: the data is valid; config is a guardrail, not
// enforcement, so fsck reports rather than rewrites.
func reportIssues(rep cache.Report) {
	for _, d := range rep.Dangling {
		fmt.Printf("dangling: %s\n", d)
	}
	for _, w := range rep.Warnings {
		fmt.Printf("warning:  %s\n", w)
	}
}
