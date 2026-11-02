#!/bin/sh
set -eu
POSIXLY_CORRECT='no bashing shell'

# Build and install git-track and its man page (ADR-0016). Installing both the
# binary (on PATH) and the man page (on MANPATH) makes 'git track' and
# 'git help track' both work. The man page is regenerated from the CLI's command
# data so it cannot drift from --help.
#
# Usage: scripts/install.sh [PREFIX]
#   PREFIX defaults to "${HOME}/.local"; binary -> PREFIX/bin,
#   man page -> PREFIX/share/man/man1.

##
# ${*} = fatal message; exits 1
die() {
  : "die(msg='${*}')"
  echo "error: ${*}" >&2
  exit 1
}

##
# ${1} = install prefix
# builds git-track, installs the binary and a freshly generated man page
install_gittrack() {
  : "install_gittrack(prefix='${1}')"
  bindir="${1}/bin"
  mandir="${1}/share/man/man1"
  mkdir -p "${bindir}" "${mandir}"

  go build -o "${bindir}/git-track" ./cmd/git-track || die 'build failed'
  # single source: regenerate the man page from the built binary (ADR-0016)
  "${bindir}/git-track" help --man > "${mandir}/git-track.1" || die 'man generation failed'

  printf 'installed:\n  %s\n  %s\n' "${bindir}/git-track" "${mandir}/git-track.1"
  printf "ensure %s is on PATH and %s on MANPATH, then:\n" "${bindir}" "${1}/share/man"
  printf '  git track --help   # usage\n  git help track     # man page\n'
}

# Run from the repository root so ./cmd/git-track resolves.
test -f go.mod || die 'run from the repository root'
install_gittrack "${1:-${HOME}/.local}"
