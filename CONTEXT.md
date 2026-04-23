# Project Context

## Synopsis

CLI email triaging tool that sorts, copies, flags, auto-archives, or deletes mail based on configurable rules (from/subject/body/date/age/folder/unread matching). Modular architecture with provider pattern for different IMAP backends (Yahoo, Gmail, etc).

## Commands

- `mailsort scan` - list unread emails  
- `mailsort scan --summary-md` - list unread emails and write a Markdown report
- `mailsort init` - create missing IMAP folders required by enabled rules
- `mailsort preview` - show which emails match which rules/actions (with summary)
- `mailsort apply` - move/copy/delete/flag/mark matching emails (dry-run with --dry-run)
- `mailsort rules` - manage rules (list, add, remove, update, enable, disable)
- `mailsort imap-debug` - inspect IMAP capabilities, mailboxes, flags, search

## Active Work

- Feature branch `feature/4-summary-md-report` implementing issue #4
- All core commands working: scan, preview, apply, rules
- Markdown summary reports are available for scan, preview, and apply via `--summary-md`
- Rule filtering: date ranges (absolute & relative), attachments, size limits
- Rule filtering: source folder, age (`older_than`/`newer_than`), unread/read status
- Rule actions: move, copy, mark as read, flag important, delete, dry-run
- Score-based auto-archive can move low-score mail from INBOX to Archive
- Rule management with enable/disable functionality

## Recent Changes

- Added `--summary-md` and `--summary-path` flags for `scan`, `preview`, and `apply`.
- Added `internal/report` Markdown renderer with run metadata, counts, important-first grouping, compact one-line email entries (`[sender] subject -> action`), action status, and warnings section.
- Wired summary generation into scan, preview, apply, including dry-run clarity for preview and `apply --dry-run`.
- Documented Markdown summary report usage and deterministic importance heuristics in README.
- Added `score`, `copy_to`, and `flag_important` rule fields.
- Added IMAP copy and important-flag operations.
- Apply now batches compatible IMAP operations by mailbox and destination/action instead of mutating one email at a time.
- Added `auto_archive` config for score-based archiving from INBOX.
- Auto-archive skips messages received in the last 24 hours and messages already flagged important.
- Added `auto_archive.date_before` for archive eligibility dates such as `-7d`.
- Added `.mailsort/rules/05-important-to-action.yaml` example to copy important mail to `To Action` and flag it.
- Added disabled `.mailsort/rules/50-score-only-example.yaml` example for score-only rules.
- Updated from/subject matching to be case-insensitive.
- Added delete action for rules via `delete: true`.
- Added source folder matching via `folder`.
- Added unread/read status matching via `unread`.
- Added age matching via `older_than` and `newer_than` durations.
- Changed preview/apply to fetch the configured mailbox plus enabled rule source folders.
- Added IMAP flag fetching so emails carry unread status.
- Added IMAP delete and mailbox-aware move/mark-as-read operations.
- Fixed dry-run detection so `mailsort apply --dry-run` works as documented.
- Updated README with the expanded rule schema and delete/dry-run examples.
- Added `.mailsort/rules/90-delete-old-linkedin.yaml` example to delete mail older than 2 days from the LinkedIn folder.
- Updated preview summary to include total planned actions and counts by action, including delete.
- Added `init` CLI command to verify required IMAP mailboxes and create missing folders.
- Added IMAP mailbox listing/creation helpers for rule destination setup.
- Fixed command error reporting so runtime failures print the actual command error.
- Added `imap-debug` CLI command with subcommands:
  - `capabilities`, `list`, `select`, `fetch-flags`, `fetch-headers`, `search`, `store`
- Added `internal/imapdebug` package for IMAP diagnostics
- Added search helpers for `ALL`, `UNSEEN`, `SINCE`, `KEYWORD`
- Added IMAP debugging documentation in `docs/imap-debug.md`
- Updated main command usage and command routing for `imap-debug`
- Updated command flow so rules directory is not required for `imap-debug`
- Added `enabled` flag to rules for enabling/disabling without deletion
- Added `enable` and `disable` subcommands to rules command
- Updated preview command to show match summary (total matched, per rule)
- Added relative date support (e.g., "-30d" for older than 30 days)
- Added email filtering criteria: date_after, date_before, has_attachments, min_size, max_size
- Added Date, Size, HasAttachments fields to Email struct
- Implemented multiple rule files in directory with priority-based ordering
- Added rule chaining support (chain field) for sequential rule application
- Built out `mailsort rules` command with list, add, remove, update subcommands
- Extended preview command to show email subject and destination folder
- Added date parsing for RFC3339, date-only (2006-01-02), and relative formats
- Created example rule files with new filtering criteria
- All tests pass, build successful
