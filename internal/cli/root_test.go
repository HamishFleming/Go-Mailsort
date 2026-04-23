package cli

import (
	"reflect"
	"testing"
	"time"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

func TestRequiredMailboxes(t *testing.T) {
	disabled := false

	cfg := &config.Config{
		Mailbox: "INBOX",
		AutoArchive: config.AutoArchiveConfig{
			Enabled: true,
			MoveTo:  "Archive",
		},
		Rules: []config.Rule{
			{Name: "github", MoveTo: "GitHub"},
			{Name: "github-duplicate", MoveTo: "GitHub"},
			{Name: "archive-source", Folder: "Archive", Delete: true},
			{Name: "action", CopyTo: "To Action", FlagImportant: true},
			{Name: "disabled", MoveTo: "Disabled", Enabled: &disabled},
			{Name: "empty"},
			{Name: "trimmed", MoveTo: " Newsletters "},
		},
	}

	got := requiredMailboxes(cfg)
	want := []string{"INBOX", "Archive", "GitHub", "To Action", "Newsletters"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requiredMailboxes() = %#v, want %#v", got, want)
	}
}

func TestSourceMailboxes(t *testing.T) {
	disabled := false

	cfg := &config.Config{
		Mailbox: "INBOX",
		Rules: []config.Rule{
			{Name: "archive", Folder: "Archive"},
			{Name: "archive-duplicate", Folder: " archive "},
			{Name: "disabled", Folder: "Trash", Enabled: &disabled},
			{Name: "default"},
		},
	}

	got := sourceMailboxes(cfg)
	want := []string{"INBOX", "Archive"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sourceMailboxes() = %#v, want %#v", got, want)
	}
}

func TestDescribeRuleAction(t *testing.T) {
	tests := []struct {
		name string
		rule *config.Rule
		want string
	}{
		{
			name: "delete",
			rule: &config.Rule{Delete: true, MoveTo: "Archive", MarkAsRead: true},
			want: "delete",
		},
		{
			name: "move and mark read",
			rule: &config.Rule{MoveTo: "Archive", MarkAsRead: true},
			want: "move to Archive, mark as read",
		},
		{
			name: "copy and flag",
			rule: &config.Rule{CopyTo: "To Action", FlagImportant: true, Score: 10},
			want: "copy to To Action, flag important",
		},
		{
			name: "no action",
			rule: &config.Rule{},
			want: "no action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeRuleAction(tt.rule); got != tt.want {
				t.Fatalf("describeRuleAction() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlanEmailActionsAutoArchive(t *testing.T) {
	cfg := &config.Config{
		Mailbox: "INBOX",
		AutoArchive: config.AutoArchiveConfig{
			Enabled:   true,
			Threshold: 0,
			MoveTo:    "Archive",
		},
	}

	email := &imapclient.Email{Mailbox: "INBOX", Uid: 1, Date: time.Now().Add(-25 * time.Hour)}
	score, actions := planEmailActions(email, nil, cfg)
	if score != 0 {
		t.Fatalf("score = %d, want 0", score)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(actions))
	}
	if actions[0].summary != "auto archive to Archive" {
		t.Fatalf("action = %q, want auto archive", actions[0].summary)
	}
}

func TestPlanEmailActionsAutoArchiveSkipsRecentMail(t *testing.T) {
	cfg := &config.Config{
		Mailbox: "INBOX",
		AutoArchive: config.AutoArchiveConfig{
			Enabled:   true,
			Threshold: 0,
			MoveTo:    "Archive",
		},
	}

	email := &imapclient.Email{Mailbox: "INBOX", Uid: 1, Date: time.Now().Add(-23 * time.Hour)}
	_, actions := planEmailActions(email, nil, cfg)
	if len(actions) != 0 {
		t.Fatalf("actions = %d, want 0 for recent mail", len(actions))
	}
}

func TestPlanEmailActionsAutoArchiveHonorsDateBefore(t *testing.T) {
	cfg := &config.Config{
		Mailbox: "INBOX",
		AutoArchive: config.AutoArchiveConfig{
			Enabled:    true,
			Threshold:  0,
			MoveTo:     "Archive",
			DateBefore: "-7d",
		},
	}

	recent := &imapclient.Email{Mailbox: "INBOX", Uid: 1, Date: time.Now().AddDate(0, 0, -3)}
	_, actions := planEmailActions(recent, nil, cfg)
	if len(actions) != 0 {
		t.Fatalf("actions = %d, want 0 for mail newer than date_before", len(actions))
	}

	old := &imapclient.Email{Mailbox: "INBOX", Uid: 2, Date: time.Now().AddDate(0, 0, -8)}
	_, actions = planEmailActions(old, nil, cfg)
	if len(actions) != 1 {
		t.Fatalf("actions = %d, want 1 for mail older than date_before", len(actions))
	}
}

func TestPlanEmailActionsAutoArchiveSkipsFlaggedMail(t *testing.T) {
	cfg := &config.Config{
		Mailbox: "INBOX",
		AutoArchive: config.AutoArchiveConfig{
			Enabled:   true,
			Threshold: 0,
			MoveTo:    "Archive",
		},
	}

	email := &imapclient.Email{Mailbox: "INBOX", Uid: 1, Date: time.Now().Add(-48 * time.Hour), Flagged: true}
	_, actions := planEmailActions(email, nil, cfg)
	if len(actions) != 0 {
		t.Fatalf("actions = %d, want 0 for flagged mail", len(actions))
	}
}

func TestPlanEmailActionsScoredImportantMailSkipsAutoArchive(t *testing.T) {
	cfg := &config.Config{
		Mailbox: "INBOX",
		AutoArchive: config.AutoArchiveConfig{
			Enabled:   true,
			Threshold: 5,
			MoveTo:    "Archive",
		},
	}

	email := &imapclient.Email{Mailbox: "INBOX", Uid: 1}
	rules := []*config.Rule{
		{Name: "important", Score: 10, CopyTo: "To Action", FlagImportant: true},
	}

	score, actions := planEmailActions(email, rules, cfg)
	if score != 10 {
		t.Fatalf("score = %d, want 10", score)
	}
	if len(actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(actions))
	}
	if actions[0].summary != "copy to To Action, flag important" {
		t.Fatalf("action = %q, want copy/flag", actions[0].summary)
	}
}

func TestRequiredMailboxesDefaultsToInbox(t *testing.T) {
	got := requiredMailboxes(&config.Config{})
	want := []string{"INBOX"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requiredMailboxes() = %#v, want %#v", got, want)
	}
}
