# Config

Config manages via git-config in `${topdir}/.gitrack/config`.
  - update_index: true/false
  - index=<file>.md: Where to place the bug index file when updating the
    database.
  - status_opened: valid opened statuses
  - status_closed: valid closed statuses
  - open_<status>_flow: <allowed next status>
  - closed_<status>_flow: <allowed_next_status>
  - severities: valid severities
  - types: valid types (bug, issue, note, etc), is this really necessary? can
    technically assume all valid directories are a valid type.
  - type_default: bug (danger of automatically opening an entry into the wrong type)

# Entry structure

Bug directory structure of `${topdir}/.gitrack/<type>/<hash>`:
  - info.md: rfc822-styled markdown document (dynamically generated)
  - head: References top-most comment, can be moved but but the path to hash to
    the entry remains unchanged.
  - tags: simple tags document, one tag per line # anything goes
  - state: (open|closed) # hard-coded
  - status: open/closed status # configurable
  - priority: severity/priority entry
  - comments/<hash>: rfc822-styled markdown document (includes Reference: hash)
  - related/: list of related entries
  - duplicate/: list of duplicate entries
  - blocks/: pointer to blocked entries
  - depends/: pointers to dependant entries
  - attachements/[<hash>]: Any file attachment (optional).  Referenced via
    rfc822 header.
