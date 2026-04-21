# mailsort Agent Guidance

- Update `CONTEXT.md` with a synopsis and recent changes after each session.
- Module: `github.com/HamishFleming/Go-Mailsort`

## Commands

- `go build ./...` - build all packages
- `go test ./...` - run all tests

## CLI Usage

Config: `.mailsort.yaml`

- `mailsort scan` - list unread emails
- `mailsort preview` - show which emails match rules
- `mailsort apply` - move matching emails (--dry-run to preview)
- `mailsort rules` - manage rules (not yet implemented)

## Architecture

- Entry: `cmd/mailsort/main.go`
- CLI: `internal/cli/root.go`
- IMAP: `internal/imapclient/`, providers in `*/yahoo`
- Rules: `internal/rules/matcher.go`

---

# Agent Team Workflow

## Remote
- `git@github.com:HamishFleming/Go-Mailsort.git`

## Workflow

### Issue Creation Agent
Creates GitHub issue from natural language task.

### Issue Resolution Agent
1. Read GitHub issue
2. Create branch: `fix/<issue-number>-<short-description>`
3. Implement fix
4. Run: `go build ./... && go test ./...`
5. Commit with meaningful message
6. Push branch
7. Create PR with issue reference

## Branch Convention
- Features: `feature/<issue-number>-<description>`
- Bugs: `fix/<issue-number>-<description>`
- PRs target: `master`

## PR Template
```markdown
## Summary
- One-liner

## Changes
- What changed

## Testing
- How verified

Closes #<issue-number>
```

## CI
Run on every PR: `go build ./... && go test ./...`