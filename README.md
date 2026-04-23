# Mailsort

CLI email triaging tool that sorts mail between IMAP folders based on configurable rules.

## Installation

```bash
go build -o mailsort ./cmd/mailsort
```

## Configuration

Create a `.mailsort.yaml` file in your working directory:

```yaml
mailbox: INBOX
rules_dir: .mailsort/rules
auto_archive:
  enabled: true
  threshold: 0
  move_to: Archive
  date_before: "-7d"
```

Create rule files in the `.mailsort/rules/` directory. Each `.yaml` file can contain one or more rules:

**`.mailsort/rules/05-important-to-action.yaml`:**
```yaml
- name: important_to_action
  priority: 5
  folder: INBOX
  subject_any:
    - urgent
    - action required
    - please review
  score: 10
  copy_to: To Action
  flag_important: true
  chain: true
```

**`.mailsort/rules/10-linkedin.yaml`:**
```yaml
- name: linkedin
  priority: 10
  from_contains:
    - linkedin.com
  subject_any:
    - is hiring
  body_any: []
  move_to: LinkedIn
  mark_as_read: false
  chain: false
```

**`.mailsort/rules/20-github.yaml`:**
```yaml
- name: github
  priority: 20
  folder: INBOX
  from_contains:
    - github.com
  subject_any:
    - GitHub
  body_any: []
  unread: true
  move_to: GitHub
  mark_as_read: true
  chain: false
```

**Delete old read mail from a folder:**
```yaml
- name: purge-old-promotions
  priority: 30
  folder: Promotions
  unread: false
  older_than: 30d
  delete: true
  chain: false
```

**Score-only example rule:**
```yaml
- name: score_only_example
  enabled: false
  priority: 50
  folder: INBOX
  from_contains: []
  subject_any: []
  body_any: []
  score: 1
  chain: true
```

### Rule Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Rule identifier (must be unique) |
| `enabled` | bool | Enable/disable rule (default: true) |
| `priority` | int | Execution order (lower = runs first) |
| `score` | int | Score contribution when the rule matches |
| `folder` | string | Source folder/mailbox to match (defaults to the configured mailbox when omitted) |
| `from_contains` | []string | Match if sender contains any of these strings |
| `subject_any` | []string | Match if subject contains any of these strings |
| `body_any` | []string | Match if body contains any of these strings |
| `date_after` | string | Match if email date is after this (RFC3339: "2024-01-15" or relative: "-7d") |
| `date_before` | string | Match if email date is before this (RFC3339: "2024-12-31" or relative: "-30d") |
| `older_than` | string | Match if email age is at least this duration (`30d`, `12h`, `90m`) |
| `newer_than` | string | Match if email age is at most this duration (`7d`, `24h`, `30m`) |
| `unread` | bool | Match unread (`true`) or read (`false`) emails |
| `has_attachments` | bool | Match if email has attachments (true/false) |
| `min_size` | uint32 | Match if email size is at least this many bytes |
| `max_size` | uint32 | Match if email size is at most this many bytes |
| `move_to` | string | Destination folder for matching emails |
| `copy_to` | string | Folder to copy matching emails to while leaving the source message in place |
| `delete` | bool | Delete matching emails instead of moving them (takes precedence over `move_to`) |
| `flag_important` | bool | Add the IMAP `\Flagged` flag to matching emails |
| `mark_as_read` | bool | Mark email as read before moving |
| `chain` | bool | If true, continue matching with next rules after this one |

### Auto Archive

The optional `auto_archive` config moves low-score mail out of the source folder, usually `INBOX`.

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Enable score-based auto-archive |
| `threshold` | int | Archive when the email score is less than or equal to this value |
| `move_to` | string | Archive destination folder (defaults to `Archive`) |
| `folder` | string | Source folder to archive from (defaults to `mailbox`) |
| `date_before` | string | Only auto-archive messages before this date, including relative values like `-7d` |

Auto-archive skips messages received in the last 24 hours and messages that are already flagged important.

## Usage

### Initialize IMAP folders
```bash
mailsort init
```

Checks the configured source mailbox and destination folders from enabled rules, creating any missing IMAP folders.

### List unread emails
```bash
mailsort scan
mailsort scan --summary-md
```

### Preview rule matches
```bash
mailsort preview
mailsort preview --summary-md
```

Shows each planned action, including deletes, and prints summary counts by action and rule.

### Apply rules (dry-run mode)
```bash
mailsort apply --dry-run
mailsort apply --dry-run --summary-md
# Also supported:
mailsort --dry-run apply
```

### Apply rules (actually move emails)
```bash
mailsort apply
mailsort apply --summary-md --summary-path ./reports/latest.md
```

Apply groups compatible IMAP operations into batches by source mailbox and destination/action, so matching emails are copied, flagged, marked read, moved, or deleted with fewer server round trips.

### Markdown summary reports

Add `--summary-md` to `scan`, `preview`, or `apply` to write a post-run Markdown report. By default reports are written to `./reports/mailsort-YYYY-MM-DD_HH-mm-ss.md`; use `--summary-path` for a stable path such as `./reports/latest.md`.

Reports include run metadata, counts, matched rules, planned or applied actions, and per-email details. The renderer lists important mail first using deterministic heuristics: urgent subject keywords, high-priority or positive-score rules, flagged mail, attachments, and unmatched human-looking senders that do not look like bulk or automated mail. `preview` and `apply --dry-run` reports state that no mail was moved or modified.

### Manage rules

List all rules (sorted by priority):
```bash
mailsort rules list
```

Add a new rule (with priority, enabled, and optional chain flag):
```bash
mailsort rules add 30 true "newsletter" "newsletter@example.com" "Newsletter,Update" "" "Newsletters" "true" [chain]
# Example with chaining enabled:
mailsort rules add 30 true "newsletter" "newsletter@example.com" "Newsletter" "" "Newsletters" "true" "true"
```

Remove a rule:
```bash
mailsort rules remove "newsletter"
```

Update an existing rule (with priority, enabled, and optional chain flag):
```bash
mailsort rules update "newsletter" 30 true "newsletter@example.com,updates@example.com" "Newsletter,Update,Alert" "" "Newsletters" "false" [chain]
```

Enable a rule:
```bash
mailsort rules enable "newsletter"
```

Disable a rule:
```bash
mailsort rules disable "newsletter"
```

### IMAP debugging toolkit

Inspect IMAP capabilities, mailboxes, flags, and search behavior:

```bash
mailsort imap-debug capabilities
mailsort imap-debug list "*"
mailsort imap-debug select INBOX
mailsort imap-debug fetch-flags 1:10
mailsort imap-debug fetch-headers 1:10
mailsort imap-debug search UNSEEN
mailsort imap-debug search SINCE 1-Feb-2026
mailsort imap-debug search KEYWORD Work
mailsort imap-debug store 1 + Work Important
```

See `docs/imap-debug.md` for a full command reference and raw IMAP examples.

## How It Works

1. **Configuration Loading**: Mailsort reads the main config from `.mailsort.yaml` which specifies the `rules_dir` directory. All `.yaml` files in that directory are loaded as rules.

2. **Rule Prioritization**: Rules are sorted by their `priority` field (lower number = runs first). Rules with the same priority maintain their file order.

3. **Email Fetching**: When you run `scan`, Mailsort fetches unread emails from the configured mailbox. When you run `preview` or `apply`, it fetches messages from the configured mailbox plus any source folders referenced by enabled rules.

4. **Rule Matching with Chaining**: The rules engine (`internal/rules/matcher.go`) evaluates each email against your rules in priority order. A rule matches if ALL specified criteria are met:
   - The sender contains any of the `from_contains` strings (if specified, case-insensitive)
   - The source mailbox matches `folder` (if specified)
   - The subject contains any of the `subject_any` strings (if specified, case-insensitive)
   - The body contains any of the `body_any` strings (if specified)
   - The email date is after `date_after` (if specified, RFC3339 or relative like "-7d")
   - The email date is before `date_before` (if specified, RFC3339 or relative like "-30d")
   - The email is older than `older_than` or newer than `newer_than` (if specified)
   - The email unread/read state matches `unread` (if specified)
   - The email has attachments matches `has_attachments` (if specified)
   - The email size is at least `min_size` bytes (if specified)
   - The email size is at most `max_size` bytes (if specified)

5. **Rule Chaining**: When a rule matches and has `chain: true`:
   - The rule's actions are applied (move email, mark as read)
   - The email (now in its new location/state) continues to be evaluated against remaining rules
   - This allows multiple rules to act on the same email in priority order
   - If `chain: false` (default), matching stops after the first rule applies

6. **Actions**: When a rule matches an email:
   - If `delete` is true, the email is marked deleted and expunged, and no move is attempted
   - If `copy_to` is set, the email is copied to that folder
   - If `flag_important` is true, the email gets the IMAP `\Flagged` flag
   - Otherwise, the email is moved to the `move_to` folder when `move_to` is set
   - If `mark_as_read` is true, the email is marked as read before moving
   - Matched rule scores are summed, and `auto_archive` moves low-score messages from INBOX when enabled, excluding mail from the last 24 hours or already flagged mail
   - With `--dry-run`, Mailsort logs the planned action without modifying messages

7. **Rule Management**: The `rules` command allows you to manage your rules directly from the CLI. Each rule is saved as an individual file in the rules directory, named using the pattern `<priority>-<name>.yaml`.

## Architecture

- **Entry point**: `cmd/mailsort/main.go`
- **CLI commands**: `internal/cli/root.go`
- **IMAP client**: `internal/imapclient/` with providers in subdirectories (e.g., `yahoo/`)
- **Rules engine**: `internal/rules/matcher.go`
- **Configuration**: `internal/config/config.go`

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create missing IMAP folders required by enabled rules |
| `scan` | List unread emails in the mailbox |
| `preview` | Show which emails match which rules/actions without modifying mail |
| `apply` | Move/delete/mark matching emails (use `--dry-run` to preview apply behavior) |
| `rules` | Manage rules (list, add, remove, update) |
