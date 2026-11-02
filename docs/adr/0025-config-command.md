# ADR-0025: `git track config` edits status workflows

## Status
accepted

## Context
Status workflows and custom statuses are stored as `config/statuses/<name>`
files whose lines are that status's allowed next statuses (ADR-0019). There is
no command to edit them yet; users would hand-edit files. The stakeholder wants
this done via a `config` command.

The grammar could mirror the `get`/`set` dotted-path accessor (ADR-0021/0022),
but workflow editing carries domain semantics — a status's next-set, and
validation that a declared transition targets a real status — that read better
as purpose-built verbs and leave room for other config domains later.

## Decision
Add `git track config` with a `status` domain for workflow editing (storage
unchanged — it edits `config/statuses/`):

- `config status list` — list all statuses (built-ins + user-defined), marking
  which are built in and showing each one's next-set.
- `config status add <name> [--next a,b,c]` — create a user-defined status with
  an optional allowed next-set. `<name>` must be a valid status name (kebab-case,
  not pure hex, not an existing status).
- `config status set-next <name> [<next>...]` — replace the allowed next-set for
  a status (built-in or user-defined). Each `<next>` must be an existing status.
  An empty next-set means unrestricted.
- `config status rm <name>` — remove a user-defined status. The three built-ins
  (`new`/`reopened`/`done`) cannot be removed; their next-sets may still be set.

Validation rules:
- transition targets must resolve to an existing status (built-in or defined) —
  no pointing a workflow at a typo.
- the built-in statuses are never removable; `rm` of one is refused.
- adding a status that already exists is refused.

The domain structure (`config <domain> <verb>`) leaves room for `config kinds
...` and other config later without overloading a flat command.

## Consequences
- Easier: workflows are editable with discoverable verbs; no hand-editing files.
- Easier: validation stops dangling transitions and typo statuses at write time.
- Easier: the `config <domain>` shape scales to future config without a flat
  grab-bag.
- Harder: slightly more command surface than reusing `get`/`set`. Accepted for
  the domain clarity and validation.
- Cost accepted: `config` and `get`/`set` both touch config files; `config` is
  the workflow-aware editor, `get`/`set`/raw-edit remain possible for anything.

## Verification
- [ ] `config status add in-review --next done` creates
      `config/statuses/in-review` containing `done`.
- [ ] `config status set-next new triaged in-progress` rewrites `new`'s next-set;
      a target that is not an existing status is refused.
- [ ] `config status list` shows built-ins and user-defined with next-sets.
- [ ] `config status rm <user-status>` deletes it; `config status rm new` (a
      built-in) is refused.
- [ ] `update --status` enforcement (ADR-0019) reflects edits made via config.
