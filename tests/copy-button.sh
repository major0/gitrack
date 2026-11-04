#!/bin/sh
set -eu
POSIXLY_CORRECT='no bashing shell'

# Real-browser test for the code-block copy button (ADR-0047). The button is a
# small inline script, which Go tests cannot run, so this builds a project that
# has every kind of <pre> the button must treat differently, generates its site,
# and drives the pages in headless Chrome:
#
#   docs/guide.md     two fenced code blocks                       -> buttons
#   main.go           a source page with numbered lines            -> a button
#   a track + board   an issue description and the board graph     -> no buttons
#
# Usage: sh tests/copy-button.sh
# Needs node and a Chrome or Chromium binary; if either is missing the test
# prints why and exits 0 (skipped). Honors ${TMPDIR:-/tmp} unless
# ${GITRACK_TEST_DIR} overrides the scratch root.

##
# ${*} = message to print to stderr
# returns: 0
error() {
  : "error(msg='${*}')"
  echo "error: ${*}" >&2
}

##
# ${*} = fatal message
# returns: does not return (exits 1)
die() {
  : "die(msg='${*}')"
  error "${*}"
  exit 1
}

##
# returns: 0 and prints the path of the first Chrome or Chromium on PATH, else 1
find_chrome() {
  : 'find_chrome()'
  for candidate in google-chrome google-chrome-stable chromium chromium-browser chrome; do
    if command -v "${candidate}" > /dev/null 2>&1; then
      command -v "${candidate}"
      return 0
    fi
  done
  return 1
}

# Decide whether to skip first, using only shell builtins, so a machine with
# neither tool exits cleanly without needing anything else on PATH.
command -v node > /dev/null 2>&1 || { echo 'skip: node not found'; exit 0; }
CHROME="$(find_chrome)" || { echo 'skip: no Chrome or Chromium found'; exit 0; }

HERE="$(cd "$(dirname "${0}")" && pwd)"
ROOT="$(dirname "${HERE}")"

BASE="${GITRACK_TEST_DIR:-${TMPDIR:-/tmp}}"
WORK="$(mktemp -d "${BASE}/gitrack-copy.XXXXXX")"
trap 'rm -rf "${WORK}"' EXIT

BIN="${WORK}/git-track"
( cd "${ROOT}" && go build -o "${BIN}" ./cmd/git-track ) || die 'build failed'

R="${WORK}/repo"
mkdir -p "${R}"
git -C "${R}" init -q -b trunk
git -C "${R}" config user.name 'Test'
git -C "${R}" config user.email 'test@example.com'
( cd "${R}" && "${BIN}" init > /dev/null )
( cd "${R}" && "${BIN}" create 'A track' > /dev/null )
( cd "${R}" && "${BIN}" set 'a-track.description' 'some preformatted description text' > /dev/null )

mkdir -p "${R}/docs"
printf '%s\n' '# Guide' '' 'Intro.' '' '## Run it' '' \
  '```sh' 'go install example.com/x@latest' 'echo "a <b> & c"' '```' '' \
  '```' 'plain block' '  with indentation' '```' > "${R}/docs/guide.md"
printf 'package main\n\nfunc main() {\n\tprintln("<hi> & bye")\n}\n' > "${R}/main.go"

SITE="${R}/.gitrack/config/site"
mkdir -p "${SITE}"
printf '%s\n' 'issues' 'docs' 'source' > "${SITE}/sections"
git -C "${R}" add -A
git -C "${R}" commit -q -m 'fixture'

( cd "${R}" && "${BIN}" web --out "${WORK}/site-on" > /dev/null ) || die 'web (button on) failed'
printf '%s\n' 'off' > "${SITE}/copy"
( cd "${R}" && "${BIN}" web --out "${WORK}/site-off" > /dev/null ) || die 'web (button off) failed'

node "${HERE}/copy-button.js" "${WORK}/site-on" "${WORK}/site-off" "${CHROME}" || die 'copy button checks failed'
echo 'DONE_COPYBUTTON'
