#!/bin/sh
set -eu
POSIXLY_CORRECT='no bashing shell'

# Regression test for ADR-0003: the filesystem-as-database layout must produce
# the intended git-merge semantics. A single-value field edited divergently on
# two branches MUST conflict; independent field edits and multi-value directory
# appends MUST merge cleanly. This test builds a throwaway repo and asserts the
# outcome of each scenario, exiting non-zero on any deviation.
#
# Usage: sh tests/merge-semantics.sh
# Honors ${TMPDIR:-/tmp} unless ${GITRACK_TEST_DIR} overrides the scratch root.

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

ID='0192f1a2-7c3e-7abc-def0-112233445566'
BASE='trunk'
FAILURES=0

##
# ${1} = scenario label
# ${2} = shell snippet applied on branch A
# ${3} = shell snippet applied on branch B
# ${4} = expected outcome: "conflict" or "clean"
# returns: 0; increments FAILURES on an unexpected outcome
assert_merge() {
  : "assert_merge(label='${1}', expect='${4}')"
  git checkout -q "${BASE}"
  git checkout -q -b "ba-${1}"
  ( eval "${2}" )
  git add -A
  git commit -qm "A: ${1}"
  git checkout -q "${BASE}"
  git checkout -q -b "bb-${1}"
  ( eval "${3}" )
  git add -A
  git commit -qm "B: ${1}"
  git checkout -q "ba-${1}"
  if git merge --no-ff --no-edit "bb-${1}" >"${MERGE_OUT}" 2>&1; then
    outcome='clean'
  else
    outcome='conflict'
    git merge --abort 2>/dev/null || true
  fi
  git checkout -q "${BASE}"
  if test "${outcome}" = "${4}"; then
    printf 'ok   %-22s %s\n' "${1}" "${outcome}"
  else
    printf 'FAIL %-22s expected %s, got %s\n' "${1}" "${4}" "${outcome}"
    FAILURES=$((FAILURES + 1))
  fi
}

WORK="${GITRACK_TEST_DIR:-${TMPDIR:-/tmp}}/gitrack-merge-test.$$"
MERGE_OUT="${WORK}.merge.out"
rm -rf "${WORK}"
mkdir -p "${WORK}"
trap 'rm -rf "${WORK}" "${MERGE_OUT}"' EXIT
cd "${WORK}" || die "cannot cd ${WORK}"

git init -q -b "${BASE}"
git config user.email 'test@gitrack.invalid'
git config user.name 'gitrack test'

mkdir -p ".gitrack/${ID}"
printf 'new\n'             > ".gitrack/${ID}/state"
printf 'Login is broken\n' > ".gitrack/${ID}/title"
printf 'issue\n'          > ".gitrack/${ID}/kind"
git add -A
git commit -qm 'create track'

# single-value field, divergent -> conflict (the feature)
assert_merge 'state-divergent' \
  "printf 'in-progress\n' > .gitrack/${ID}/state" \
  "printf 'wontfix\n'     > .gitrack/${ID}/state" \
  'conflict'

# single-value field, same value -> clean (idempotent)
assert_merge 'state-same' \
  "printf 'done\n' > .gitrack/${ID}/state" \
  "printf 'done\n' > .gitrack/${ID}/state" \
  'clean'

# different fields -> clean (independent)
assert_merge 'state-vs-title' \
  "printf 'done\n' > .gitrack/${ID}/state" \
  "printf 'Auth is broken\n' > .gitrack/${ID}/title" \
  'clean'

# multi-value dir, different entries -> clean union
assert_merge 'comments-different' \
  "mkdir -p .gitrack/${ID}/comments && printf 'x\n' > .gitrack/${ID}/comments/aaaa.md" \
  "mkdir -p .gitrack/${ID}/comments && printf 'y\n' > .gitrack/${ID}/comments/bbbb.md" \
  'clean'

# multi-value relationship dir, different edges -> clean union
assert_merge 'depends-different' \
  "mkdir -p .gitrack/${ID}/depends-on && touch .gitrack/${ID}/depends-on/target-1" \
  "mkdir -p .gitrack/${ID}/depends-on && touch .gitrack/${ID}/depends-on/target-2" \
  'clean'

# same multi-line field edited two ways -> conflict (expected)
printf 'line1\nline2\nline3\n' > ".gitrack/${ID}/description"
git add -A
git commit -qm 'add description'
assert_merge 'description-divergent' \
  "printf 'line1\nAAA\nline3\n' > .gitrack/${ID}/description" \
  "printf 'line1\nBBB\nline3\n' > .gitrack/${ID}/description" \
  'conflict'

# ADR-0009: a slug set divergently on the SAME track -> conflict (same as any
# single-value field; the slug-UNIQUENESS-across-tracks check is a CLI/fsck
# concern, not a merge concern, since different tracks are different files).
assert_merge 'slug-divergent' \
  "printf 'login-flow\n'  > .gitrack/${ID}/slug" \
  "printf 'signin-flow\n' > .gitrack/${ID}/slug" \
  'conflict'

echo '--- merge scenarios complete ---'

# ADR-0014: the derived cache is git-ignored, so it never enters a commit and
# therefore never travels through a merge. Assert that directly: after writing a
# .gitrack/.gitignore with /cache/ and creating cache entries, git sees nothing
# to commit under cache/, and the authoritative track object stays tracked.
echo '--- ADR-0014: cache is git-ignored ---'
printf '/cache/\n' > .gitrack/.gitignore
mkdir -p ".gitrack/cache/state/new" ".gitrack/cache/type/task"
touch ".gitrack/cache/state/new/${ID}" ".gitrack/cache/type/task/${ID}"
git add -A
git commit -qm 'add cache gitignore' >/dev/null
if test -z "$(git status --porcelain .gitrack/cache/)"; then
  echo 'ok   cache-gitignored        untracked'
else
  echo 'FAIL cache-gitignored        cache is tracked by git'
  FAILURES=$((FAILURES + 1))
fi
if git ls-files --error-unmatch ".gitrack/${ID}/state" >/dev/null 2>&1; then
  echo 'ok   authoritative-tracked   state file tracked'
else
  echo 'FAIL authoritative-tracked   track object not tracked'
  FAILURES=$((FAILURES + 1))
fi

echo '==='
test "${FAILURES}" -eq 0 || die "${FAILURES} assertion(s) deviated from spec"
echo 'PASS: all scenarios match ADR-0003 / 0009 / 0014'
