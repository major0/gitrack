package main

import (
	"fmt"
	"strings"
	"time"
)

// manPage renders the git-track(1) man page as roff, from the same command data
// that drives --help (ADR-0016: one source, no drift). Emitted via
// `git track help --man` and captured into man/git-track.1 for packaging.
func manPage() string {
	var b strings.Builder
	date := time.Now().UTC().Format("2006-01-02")

	// .TH title section date source manual
	fmt.Fprintf(&b, ".TH GIT-TRACK 1 %q \"gitrack\" \"Git Track Manual\"\n", date)

	b.WriteString(".SH NAME\n")
	fmt.Fprintf(&b, "git-track \\- %s\n", roffEscape(tagline))

	b.WriteString(".SH SYNOPSIS\n")
	b.WriteString(".B git track\n")
	b.WriteString(".I command\n")
	b.WriteString("[\\fIargs\\fR]\n")

	b.WriteString(".SH DESCRIPTION\n")
	b.WriteString("Git Track is a distributed issue/task/note tracker whose state lives in the\n")
	b.WriteString("git working tree and merges as part of an ordinary \\fBgit merge\\fR. It installs\n")
	b.WriteString("as \\fBgit-track\\fR on \\fBPATH\\fR and runs as the \\fBgit track\\fR subcommand.\n")

	b.WriteString(".SH COMMANDS\n")
	for _, c := range commands {
		synopsis := c.Name
		if c.Args != "" {
			synopsis += " " + c.Args
		}
		// .TP = tagged paragraph: bold synopsis, then the summary.
		b.WriteString(".TP\n")
		fmt.Fprintf(&b, ".B %s\n", roffEscape(synopsis))
		fmt.Fprintf(&b, "%s\n", roffEscape(c.Summary))
	}

	b.WriteString(".SH MODEL\n")
	b.WriteString(roffEscape(strings.ReplaceAll(overview, "\n", " ")))
	b.WriteString("\n")

	b.WriteString(".SH FILES\n")
	b.WriteString(".TP\n")
	b.WriteString(".B .gitrack/\n")
	b.WriteString("The tracker tree in the working directory: one directory per track keyed by\n")
	b.WriteString("UUID, config under \\fB.gitrack/config/\\fR, and a git-ignored derived cache\n")
	b.WriteString("under \\fB.gitrack/cache/\\fR. See the on-disk format specification.\n")

	b.WriteString(".SH SEE ALSO\n")
	b.WriteString(".BR git (1)\n")

	b.WriteString(".SH AUTHOR\n")
	b.WriteString("Mark Ferrell.\n")

	return b.String()
}

// roffEscape escapes the characters roff treats specially. A leading dot or
// apostrophe would start a request; backslash is the roff escape char.
func roffEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\(rs`)
	s = strings.ReplaceAll(s, "-", `\-`)
	if strings.HasPrefix(s, ".") || strings.HasPrefix(s, "'") {
		s = `\&` + s
	}
	return s
}
