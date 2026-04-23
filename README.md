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

rules:
  - name: linkedin
    from_contains:
      - linkedin.com
    subject_any:
      - is hiring
    move_to: LinkedIn
    mark_as_read: false

  - name: github
    from_contains:
      - github.com
    subject_any:
      - GitHub
    move_to: GitHub
    mark_as_read: true
```

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

List all rules:
```bash
mailsort rules list
```

Add a new rule:
```bash
mailsort rules add "newsletter" "newsletter@example.com" "Newsletter,Update" "" "Newsletters" "true"
```

Remove a rule:
```bash
mailsort rules remove "newsletter"
```

Update an existing rule:
```bash
mailsort rules update "newsletter" "newsletter@example.com,updates@example.com" "Newsletter,Update,Alert" "" "Newsletters" "false"
```

## How It Works

1. **Configuration Loading**: Mailsort reads rules from `.mailsort.yaml` which defines matching criteria (from, subject, body) and actions (move to folder, mark as read).

2. **Email Fetching**: When you run a command, Mailsort connects to your IMAP server (currently supports Yahoo) and fetches unread emails from the specified mailbox.

3. **Rule Matching**: The rules engine (`internal/rules/matcher.go`) evaluates each email against your configured rules in order. A rule matches if:
   - The sender contains any of the `from_contains` strings (if specified)
   - The subject contains any of the `subject_any` strings (if specified)
   - The body contains any of the `body_any` strings (if specified)

4. **Actions**: When a rule matches an email:
   - The email is moved to the `move_to` folder
   - If `mark_as_read` is true, the email is marked as read

5. **Rule Management**: The `rules` command allows you to manage your rules directly from the CLI without manually editing the YAML file. Changes are saved immediately to `.mailsort.yaml`.

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
