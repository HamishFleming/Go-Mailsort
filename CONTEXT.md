# Project Context

## Synopsis

CLI email triaging tool that sorts mail between IMAP folders based on configurable rules (from/subject/body matching). Modular architecture with provider pattern for different IMAP backends (Yahoo, Gmail, etc).

## Commands

- `mailsort scan` - list unread emails  
- `mailsort init` - create missing IMAP folders required by enabled rules
- `mailsort preview` - show which emails match which rules (with summary)
- `mailsort apply` - move matching emails (dry-run with --dry-run)
- `mailsort rules` - manage rules (list, add, remove, update, enable, disable)
- `mailsort imap-debug` - inspect IMAP capabilities, mailboxes, flags, search

## Active Work

- Feature branch `feature/imap-debug-toolkit` implementing issue #3
- All core commands working: scan, preview, apply, rules
- Rule filtering: date ranges (absolute & relative), attachments, size limits
- Rule management with enable/disable functionality

## Recent Changes

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
