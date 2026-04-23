# Project Context

## Synopsis

CLI email triaging tool that sorts mail between IMAP folders based on configurable rules (from/subject/body matching). Modular architecture with provider pattern for different IMAP backends (Yahoo, Gmail, etc).

## Commands

- `mailsort scan` - list unread emails  
- `mailsort preview` - show which emails match which rules
- `mailsort apply` - move matching emails (dry-run with --dry-run)
- `mailsort rules` - manage rules (not yet implemented)

## Active Work

- All core commands working: scan, preview, apply
- Apply attempts to move emails; fails if destination folders don't exist

## Recent Changes

- Added email filtering criteria: date ranges, attachments, size limits
- Added Date, Size, HasAttachments fields to Email struct
- Added date_after, date_before, has_attachments, min_size, max_size to Rule
- Fixed date parsing to handle both RFC3339 and date-only formats (2006-01-02)
- Added parseDate() function with support for multiple date formats
- Added relative date support (e.g., "-30d" for older than 30 days)
- Updated matcher to check new filter criteria
- Created example rule file with new filtering criteria (30-filters-example.yaml)
- All core commands working: scan, preview, apply, rules
- Implemented multiple rule files in directory with priority-based ordering
- Added rule chaining support (chain field) for sequential rule application
- Built out `mailsort rules` command with list, add, remove, update subcommands
- Extended preview command to show email subject and destination folder