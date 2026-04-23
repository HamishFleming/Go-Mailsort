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
```

Create rule files in the `.mailsort/rules/` directory. Each `.yaml` file can contain one or more rules:

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
  from_contains:
    - github.com
  subject_any:
    - GitHub
  body_any: []
  move_to: GitHub
  mark_as_read: true
  chain: false
```

### Rule Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Rule identifier (must be unique) |
| `enabled` | bool | Enable/disable rule (default: true) |
| `priority` | int | Execution order (lower = runs first) |
| `from_contains` | []string | Match if sender contains any of these strings |
| `subject_any` | []string | Match if subject contains any of these strings |
| `body_any` | []string | Match if body contains any of these strings |
| `date_after` | string | Match if email date is after this (RFC3339: "2024-01-15" or relative: "-7d") |
| `date_before` | string | Match if email date is before this (RFC3339: "2024-12-31" or relative: "-30d") |
| `has_attachments` | bool | Match if email has attachments (true/false) |
| `min_size` | uint32 | Match if email size is at least this many bytes |
| `max_size` | uint32 | Match if email size is at most this many bytes |
| `move_to` | string | Destination folder for matching emails |
| `mark_as_read` | bool | Mark email as read after moving |
| `chain` | bool | If true, continue matching with next rules after this one |

## Usage

### List unread emails
```bash
mailsort scan
```

### Preview rule matches
```bash
mailsort preview
```

### Apply rules (dry-run mode)
```bash
mailsort apply --dry-run
```

### Apply rules (actually move emails)
```bash
mailsort apply
```

### Manage rules

List all rules (sorted by priority):
```bash
mailsort rules list
```

Add a new rule (with priority and optional chain flag):
```bash
mailsort rules add 30 "newsletter" "newsletter@example.com" "Newsletter,Update" "" "Newsletters" "true" [chain]
# Example with chaining enabled:
mailsort rules add 30 "newsletter" "newsletter@example.com" "Newsletter" "" "Newsletters" "true" "true"
```

Remove a rule:
```bash
mailsort rules remove "newsletter"
```

Update an existing rule (with priority and optional chain flag):
```bash
mailsort rules update "newsletter" 30 "newsletter@example.com,updates@example.com" "Newsletter,Update,Alert" "" "Newsletters" "false" [chain]
```

## How It Works

1. **Configuration Loading**: Mailsort reads the main config from `.mailsort.yaml` which specifies the `rules_dir` directory. All `.yaml` files in that directory are loaded as rules.

2. **Rule Prioritization**: Rules are sorted by their `priority` field (lower number = runs first). Rules with the same priority maintain their file order.

3. **Email Fetching**: When you run a command, Mailsort connects to your IMAP server (currently supports Yahoo) and fetches unread emails from the specified mailbox.

4. **Rule Matching with Chaining**: The rules engine (`internal/rules/matcher.go`) evaluates each email against your rules in priority order. A rule matches if ALL specified criteria are met:
   - The sender contains any of the `from_contains` strings (if specified)
   - The subject contains any of the `subject_any` strings (if specified)
   - The body contains any of the `body_any` strings (if specified)
   - The email date is after `date_after` (if specified, RFC3339 or relative like "-7d")
   - The email date is before `date_before` (if specified, RFC3339 or relative like "-30d")
   - The email has attachments matches `has_attachments` (if specified)
   - The email size is at least `min_size` bytes (if specified)
   - The email size is at most `max_size` bytes (if specified)

5. **Rule Chaining**: When a rule matches and has `chain: true`:
   - The rule's actions are applied (move email, mark as read)
   - The email (now in its new location/state) continues to be evaluated against remaining rules
   - This allows multiple rules to act on the same email in priority order
   - If `chain: false` (default), matching stops after the first rule applies

6. **Actions**: When a rule matches an email:
   - The email is moved to the `move_to` folder
   - If `mark_as_read` is true, the email is marked as read

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
| `scan` | List unread emails in the mailbox |
| `preview` | Show which emails match which rules (without moving) |
| `apply` | Move matching emails to their destination folders (use `--dry-run` to preview) |
| `rules` | Manage rules (list, add, remove, update) |
