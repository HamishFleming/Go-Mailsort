package report

import (
	"strings"
	"testing"
	"time"

	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

func TestRenderMarkdownOrdersImportantFirstAndShowsDryRun(t *testing.T) {
	result := Result{
		Command:    "apply",
		Mailbox:    "INBOX",
		Timestamp:  time.Date(2026, 4, 23, 18, 30, 0, 0, time.Local),
		DryRun:     true,
		OutputPath: "reports/latest.md",
		Emails: []EmailResult{
			{
				Email: imapclient.Email{
					Uid:     1,
					Mailbox: "INBOX",
					From:    "updates@example.com",
					Subject: "Weekly Product Update",
					Date:    time.Date(2026, 4, 23, 17, 0, 0, 0, time.Local),
				},
				Rules: []RuleInfo{{Name: "newsletter", Priority: 40, MoveTo: "Newsletters"}},
				Actions: []ActionInfo{{
					Summary: "move to Newsletters",
				}},
			},
			{
				Email: imapclient.Email{
					Uid:            2,
					Mailbox:        "INBOX",
					From:           "jane@example.com",
					Subject:        "Urgent contract question",
					Date:           time.Date(2026, 4, 23, 18, 0, 0, 0, time.Local),
					HasAttachments: true,
				},
				Score: 10,
				Rules: []RuleInfo{{Name: "vip", Priority: 5, MoveTo: "Important", MarkAsRead: true}},
				Actions: []ActionInfo{{
					Summary: "move to Important, mark as read",
				}},
			},
		},
	}

	md := RenderMarkdown(result)
	mustContain(t, md, "- Report mode: dry-run, no mail was moved or modified")
	mustContain(t, md, "- Matches: 2")
	mustContain(t, md, "- Actions taken: 0")
	mustContain(t, md, "- Planned actions: 2")
	mustContain(t, md, "## Important")
	mustContain(t, md, "- [jane@example.com] Urgent contract question -> move to Important, mark as read")
	mustContain(t, md, "## Newsletters / Bulk")
	mustContain(t, md, "- [updates@example.com] Weekly Product Update -> move to Newsletters")

	if strings.Index(md, "## Important") > strings.Index(md, "## Rule Matches") {
		t.Fatalf("important section should render before rule matches")
	}
}

func TestRenderMarkdownShowsRulesWhenThereIsNoAction(t *testing.T) {
	result := Result{
		Command:   "preview",
		Mailbox:   "INBOX",
		Timestamp: time.Date(2026, 4, 23, 18, 30, 0, 0, time.Local),
		DryRun:    true,
		Emails: []EmailResult{
			{
				Email: imapclient.Email{
					Uid:     3,
					Mailbox: "INBOX",
					From:    "alerts@example.com",
					Subject: "System notice",
				},
				Rules: []RuleInfo{{Name: "score-only", Priority: 20}},
			},
		},
	}

	md := RenderMarkdown(result)
	mustContain(t, md, "- [alerts@example.com] System notice -> matched score-only")
}

func TestDefaultPath(t *testing.T) {
	now := time.Date(2026, 4, 23, 18, 30, 0, 0, time.UTC)
	got := DefaultPath(now)
	want := "reports/mailsort-2026-04-23_18-30-00.md"
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func mustContain(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected markdown to contain %q\n%s", want, text)
	}
}
