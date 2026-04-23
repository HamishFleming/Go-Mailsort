package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

type Result struct {
	Command    string
	Mailbox    string
	Timestamp  time.Time
	DryRun     bool
	OutputPath string
	Emails     []EmailResult
	Warnings   []string
}

type EmailResult struct {
	Email   imapclient.Email
	Score   int
	Rules   []RuleInfo
	Actions []ActionInfo
}

type RuleInfo struct {
	Name          string
	Priority      int
	MoveTo        string
	CopyTo        string
	Delete        bool
	MarkAsRead    bool
	FlagImportant bool
	Chain         bool
}

type ActionInfo struct {
	Summary string
	Rule    string
	Applied bool
}

func DefaultPath(now time.Time) string {
	return filepath.Join("reports", "mailsort-"+now.Format("2006-01-02_15-04-05")+".md")
}

func WriteMarkdown(result Result, path string) (string, error) {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}
	if strings.TrimSpace(path) == "" {
		path = DefaultPath(result.Timestamp)
	}
	result.OutputPath = path

	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create report directory: %w", err)
		}
	}

	data := []byte(RenderMarkdown(result))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

func RenderMarkdown(result Result) string {
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Mailsort Run Summary\n\n")
	fmt.Fprintf(&b, "- Time: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- Command: %s\n", valueOr(result.Command, "unknown"))
	fmt.Fprintf(&b, "- Mailbox: %s\n", valueOr(result.Mailbox, "INBOX"))
	fmt.Fprintf(&b, "- Emails scanned: %d\n", len(result.Emails))
	fmt.Fprintf(&b, "- Matches: %d\n", matchedCount(result.Emails))
	fmt.Fprintf(&b, "- Actions taken: %d\n", actionCount(result.Emails, true))
	fmt.Fprintf(&b, "- Planned actions: %d\n", actionCount(result.Emails, false))
	fmt.Fprintf(&b, "- Moved/deleted: %d\n", moveDeleteCount(result.Emails, true))
	fmt.Fprintf(&b, "- Marked read: %d\n", markReadCount(result.Emails, true))
	fmt.Fprintf(&b, "- Unmatched: %d\n", unmatchedCount(result.Emails))
	fmt.Fprintf(&b, "- Output path: %s\n", valueOr(result.OutputPath, "not written"))
	if result.DryRun {
		fmt.Fprintf(&b, "- Report mode: dry-run, no mail was moved or modified\n")
	} else {
		fmt.Fprintf(&b, "- Report mode: live\n")
	}
	fmt.Fprintf(&b, "\nImportance heuristic: urgent subjects, high-priority or positive-score rules, flagged mail, attachments, and non-bulk unmatched human-looking senders are listed first.\n")

	sections := buildSections(result.Emails)
	for _, section := range sections {
		fmt.Fprintf(&b, "\n## %s\n\n", section.title)
		if len(section.emails) == 0 {
			fmt.Fprintf(&b, "- None\n")
			continue
		}
		for i, email := range section.emails {
			writeEmail(&b, i+1, email, result.DryRun)
		}
	}

	fmt.Fprintf(&b, "\n## Errors / Warnings\n\n")
	if len(result.Warnings) == 0 {
		fmt.Fprintf(&b, "- None\n")
	} else {
		for _, warning := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", oneLine(warning))
		}
	}

	return b.String()
}

type section struct {
	title  string
	emails []EmailResult
}

func buildSections(emails []EmailResult) []section {
	sections := []section{
		{title: "Important"},
		{title: "Rule Matches"},
		{title: "Uncategorized"},
		{title: "Newsletters / Bulk"},
		{title: "Notifications / Automated"},
	}

	for _, email := range emails {
		idx := sectionIndex(email)
		sections[idx].emails = append(sections[idx].emails, email)
	}
	for i := range sections {
		sortEmailResults(sections[i].emails)
	}
	return sections
}

func sectionIndex(email EmailResult) int {
	if isImportant(email) {
		return 0
	}
	if isNewsletter(email) {
		return 3
	}
	if isAutomated(email) {
		return 4
	}
	if len(email.Rules) > 0 || len(email.Actions) > 0 {
		return 1
	}
	return 2
}

func isImportant(email EmailResult) bool {
	subject := strings.ToLower(email.Email.Subject)
	for _, keyword := range []string{"urgent", "asap", "important", "deadline", "action required", "security", "invoice", "payment"} {
		if strings.Contains(subject, keyword) {
			return true
		}
	}
	if email.Score > 0 || email.Email.Flagged || email.Email.HasAttachments {
		return true
	}
	for _, rule := range email.Rules {
		if rule.Priority > 0 && rule.Priority <= 10 {
			return true
		}
	}
	return len(email.Rules) == 0 && looksHuman(email.Email.From) && !isNewsletter(email) && !isAutomated(email)
}

func isNewsletter(email EmailResult) bool {
	text := strings.ToLower(email.Email.From + " " + email.Email.Subject + " " + strings.Join(ruleNames(email.Rules), " "))
	for _, keyword := range []string{"newsletter", "digest", "weekly update", "marketing", "unsubscribe", "campaign", "product update"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func isAutomated(email EmailResult) bool {
	from := strings.ToLower(email.Email.From)
	subject := strings.ToLower(email.Email.Subject)
	for _, keyword := range []string{"no-reply", "noreply", "notification", "notifications@", "alerts@", "do-not-reply", "receipt", "statement"} {
		if strings.Contains(from, keyword) || strings.Contains(subject, keyword) {
			return true
		}
	}
	return false
}

func looksHuman(from string) bool {
	from = strings.ToLower(strings.TrimSpace(from))
	if from == "" || !strings.Contains(from, "@") {
		return false
	}
	if strings.Contains(from, "+") {
		return false
	}
	for _, keyword := range []string{"no-reply", "noreply", "notifications", "support", "team", "hello", "info", "admin", "sales"} {
		if strings.Contains(from, keyword) {
			return false
		}
	}
	return true
}

func sortEmailResults(emails []EmailResult) {
	sort.SliceStable(emails, func(i, j int) bool {
		if !emails[i].Email.Date.Equal(emails[j].Email.Date) {
			return emails[i].Email.Date.After(emails[j].Email.Date)
		}
		if emails[i].Email.Subject != emails[j].Email.Subject {
			return emails[i].Email.Subject < emails[j].Email.Subject
		}
		return emails[i].Email.Uid < emails[j].Email.Uid
	})
}

func writeEmail(b *strings.Builder, index int, result EmailResult, dryRun bool) {
	subject := valueOr(result.Email.Subject, "(no subject)")
	fmt.Fprintf(b, "### %d. %s\n", index, oneLine(subject))
	fmt.Fprintf(b, "- From: %s\n", valueOr(result.Email.From, "unknown"))
	fmt.Fprintf(b, "- Date: %s\n", formatDate(result.Email.Date))
	fmt.Fprintf(b, "- Folder: %s\n", valueOr(result.Email.Mailbox, "INBOX"))
	fmt.Fprintf(b, "- UID: %d\n", result.Email.Uid)
	fmt.Fprintf(b, "- Score: %d\n", result.Score)
	if result.Email.HasAttachments {
		fmt.Fprintf(b, "- Attachments: yes\n")
	}
	if len(result.Rules) == 0 {
		fmt.Fprintf(b, "- Rule: none\n")
	} else {
		for _, rule := range result.Rules {
			fmt.Fprintf(b, "- Rule: %s (priority %d, move_to: %s, copy_to: %s, delete: %t, mark_as_read: %t, flag_important: %t, chain: %t)\n",
				valueOr(rule.Name, "unnamed"), rule.Priority, valueOr(rule.MoveTo, "none"), valueOr(rule.CopyTo, "none"), rule.Delete, rule.MarkAsRead, rule.FlagImportant, rule.Chain)
		}
	}
	if len(result.Actions) == 0 {
		fmt.Fprintf(b, "- Action: none\n")
	} else {
		label := "Action"
		if dryRun {
			label = "Planned action"
		}
		for _, action := range result.Actions {
			state := "planned"
			if action.Applied {
				state = "applied"
			}
			fmt.Fprintf(b, "- %s: %s (%s)\n", label, valueOr(action.Summary, "none"), state)
		}
	}
	fmt.Fprintf(b, "- Snippet: %s\n\n", snippet(result.Email.Body))
}

func matchedCount(emails []EmailResult) int {
	count := 0
	for _, email := range emails {
		if len(email.Rules) > 0 || len(email.Actions) > 0 {
			count++
		}
	}
	return count
}

func unmatchedCount(emails []EmailResult) int {
	return len(emails) - matchedCount(emails)
}

func actionCount(emails []EmailResult, appliedOnly bool) int {
	count := 0
	for _, email := range emails {
		for _, action := range email.Actions {
			if !appliedOnly || action.Applied {
				count++
			}
		}
	}
	return count
}

func moveDeleteCount(emails []EmailResult, appliedOnly bool) int {
	count := 0
	for _, email := range emails {
		for _, action := range email.Actions {
			summary := strings.ToLower(action.Summary)
			if (strings.Contains(summary, "move to ") || strings.Contains(summary, "auto archive to ") || summary == "delete") && (!appliedOnly || action.Applied) {
				count++
			}
		}
	}
	return count
}

func markReadCount(emails []EmailResult, appliedOnly bool) int {
	count := 0
	for _, email := range emails {
		for _, action := range email.Actions {
			if strings.Contains(strings.ToLower(action.Summary), "mark as read") && (!appliedOnly || action.Applied) {
				count++
			}
		}
	}
	return count
}

func ruleNames(rules []RuleInfo) []string {
	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		names = append(names, rule.Name)
	}
	return names
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02 15:04")
}

func snippet(body string) string {
	body = strings.TrimSpace(oneLine(body))
	if body == "" {
		return "not available"
	}
	if len(body) > 180 {
		return body[:177] + "..."
	}
	return body
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func valueOr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
