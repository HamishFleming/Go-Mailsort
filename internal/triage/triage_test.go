package triage

import (
	"testing"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
	"github.com/HamishFleming/Go-Mailsort/internal/rules"
)

func TestNewTriage(t *testing.T) {
	tests := []struct {
		name         string
		emails       []imapclient.Email
		rules        []config.Rule
		wantCount    int
		wantMatched int
	}{
		{
			name: "all matched",
			emails: []imapclient.Email{
				{Uid: 1, From: "test@example.com", Subject: "Alert"},
				{Uid: 2, From: "test@example.com", Subject: "Alert"},
			},
			rules: []config.Rule{
				{Name: "alerts", SubjectAny: []string{"Alert"}, MoveTo: "Alerts"},
			},
			wantCount:    2,
			wantMatched: 2,
		},
		{
			name: "none matched",
			emails: []imapclient.Email{
				{Uid: 1, From: "test@example.com", Subject: "Regular"},
				{Uid: 2, From: "test@example.com", Subject: "Regular"},
			},
			rules: []config.Rule{
				{Name: "alerts", SubjectAny: []string{"Alert"}, MoveTo: "Alerts"},
			},
			wantCount:    2,
			wantMatched: 0,
		},
		{
			name: "mixed match and unmatched",
			emails: []imapclient.Email{
				{Uid: 1, From: "test@example.com", Subject: "Alert"},
				{Uid: 2, From: "test@example.com", Subject: "Regular"},
				{Uid: 3, From: "test@example.com", Subject: "Warning"},
			},
			rules: []config.Rule{
				{Name: "alerts", SubjectAny: []string{"Alert", "Warning"}, MoveTo: "Alerts"},
			},
			wantCount:    3,
			wantMatched: 2,
		},
		{
			name: "first rule wins",
			emails: []imapclient.Email{
				{Uid: 1, From: "test@example.com", Subject: "Alert"},
			},
			rules: []config.Rule{
				{Name: "rule1", SubjectAny: []string{"Alert"}, MoveTo: "folder1"},
				{Name: "rule2", SubjectAny: []string{"Alert"}, MoveTo: "folder2"},
			},
			wantCount:    1,
			wantMatched: 1,
		},
		{
			name: "empty emails",
			emails:       []imapclient.Email{},
			rules:        []config.Rule{{Name: "rule1", MoveTo: "folder1"}},
			wantCount:    0,
			wantMatched: 0,
		},
		{
			name: "empty rules - all unmatched",
			emails: []imapclient.Email{
				{Uid: 1, From: "test@example.com", Subject: "Subject"},
			},
			rules:        []config.Rule{},
			wantCount:    1,
			wantMatched: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := rules.NewMatcher(tt.rules)
			got := NewTriage(tt.emails, matcher)

			if len(got) != tt.wantCount {
				t.Errorf("NewTriage() returned %d messages, want %d", len(got), tt.wantCount)
			}

			matched := 0
			for _, m := range got {
				if len(m.Rules) > 0 {
					matched++
				}
			}
			if matched != tt.wantMatched {
				t.Errorf("matched count = %d, want %d", matched, tt.wantMatched)
			}
		})
	}
}