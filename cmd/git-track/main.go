// Command git-track is the gitrack CLI, installed on PATH so that git's native
// subcommand dispatch runs it as "git track" (ADR-0015/0016). This is the first
// vertical slice: init, new, show, list, fsck.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/major0/gitrack/internal/cache"
	"github.com/major0/gitrack/internal/core"
)

const usage = `git-track - distributed tracker whose state merges with your code

usage: git track <command> [args]

commands:
  init                 initialize a tracker in the current repository
  new [-k kind] <title>  create a track, print its id
  show <ref>           display a track (ref: id, short id, or slug)
  list [-s state] [-k kind]  list tracks
  fsck [--write]       verify (or rebuild) the derived cache

See docs/FORMAT.md for the on-disk format.`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "git-track:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Println(usage)
		return nil
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "init":
		return cmdInit()
	case "new":
		return cmdNew(rest)
	case "show":
		return cmdShow(rest)
	case "list", "ls":
		return cmdList(rest)
	case "fsck":
		return cmdFsck(rest)
	default:
		return fmt.Errorf("unknown command %q (try 'git track --help')", cmd)
	}
}

func openRepo() (*core.Repo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return core.Open(wd)
}

func cmdInit() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	repo, err := core.Init(wd)
	if err != nil {
		return err
	}
	fmt.Printf("initialized gitrack tracker in %s/%s\n", repo.Root, core.Dir)
	return nil
}

func cmdNew(args []string) error {
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
		return fmt.Errorf("new requires a title")
	}
	repo, err := openRepo()
	if err != nil {
		return err
	}
	id, err := repo.Create(core.NewTrackOpts{Kind: kind, Title: strings.Join(title, " ")})
	if err != nil {
		return err
	}
	if _, err := cache.Rebuild(repo); err != nil {
		return fmt.Errorf("updating cache: %w", err)
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
	closed, err := repo.IsClosed(id)
	if err != nil {
		return err
	}
	fmt.Printf("id:      %s\n", t.ID)
	fmt.Printf("slug:    %s\n", t.Slug)
	fmt.Printf("kind:    %s\n", t.Kind)
	fmt.Printf("title:   %s\n", t.Title)
	state := t.State
	if closed {
		state += " (closed)"
	}
	fmt.Printf("state:   %s\n", state)
	for _, rel := range core.RelationKinds {
		if targets := t.Relations[rel]; len(targets) > 0 {
			fmt.Printf("%s: %s\n", rel, strings.Join(targets, ", "))
		}
	}
	if t.Description != "" {
		fmt.Printf("\n%s\n", t.Description)
	}
	return nil
}

func cmdList(args []string) error {
	var wantState, wantKind string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-s", "--state":
			if i+1 >= len(args) {
				return fmt.Errorf("-s requires a state")
			}
			i++
			wantState = args[i]
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
		if wantKind != "" && t.Kind != wantKind {
			continue
		}
		short := strings.ReplaceAll(id, "-", "")[:8]
		fmt.Printf("%s  %-11s %-8s %s\n", short, t.State, t.Kind, t.Title)
	}
	return nil
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
		reportDangling(rep)
		return nil
	}
	rep, err := cache.Verify(repo)
	if err != nil {
		return err
	}
	if rep.Clean() {
		fmt.Println("cache is clean")
		return nil
	}
	for _, m := range rep.Missing {
		fmt.Printf("missing: %s\n", m)
	}
	for _, e := range rep.Extra {
		fmt.Printf("extra:   %s\n", e)
	}
	reportDangling(rep)
	fmt.Fprintln(os.Stderr, "\nrun 'git track fsck --write' to rebuild")
	// Exit nonzero on drift so CI and scripts can gate on a clean cache,
	// matching 'git fsck' and linter conventions.
	return errDrift
}

// errDrift signals that a read-only fsck found the cache out of date.
var errDrift = fmt.Errorf("cache is out of date")

func reportDangling(rep cache.Report) {
	for _, d := range rep.Dangling {
		fmt.Printf("dangling: %s\n", d)
	}
}
