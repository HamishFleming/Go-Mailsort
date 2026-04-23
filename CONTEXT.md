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

- Implemented multiple rule files in directory with priority-based ordering
- Added rule chaining support (chain field) for sequential rule application
- Updated config to load rules from `.mailsort/rules/` directory
- Added `priority` field to rules for execution order
- Updated `Match()` to return `[]*config.Rule` for chaining support
- Updated triage package to handle multiple matched rules
- All core commands working: scan, preview, apply, rules
- Built out `mailsort rules` command with list, add, remove, update subcommands
- Added `config.LoadRulesFromDir()` for directory-based rule loading
- Added `config.LoadMainConfig()` for main config file
- Updated README with new configuration format and chaining documentation