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

- Built out `mailsort rules` command with list, add, remove, update subcommands
- Added `config.Save` function to persist rule changes to YAML
- Added helper functions: `splitCsv` for parsing comma-separated values, `parseBool` for boolean parsing
- All core commands working: scan, preview, apply, rules
- Added test suite for rules, triage, and config packages
- Implemented Apply command with --dry-run flag
- Implemented Preview command showing rule matches
- Implemented provider pattern for Yahoo IMAP
- Scan command working (connects and lists unread emails)