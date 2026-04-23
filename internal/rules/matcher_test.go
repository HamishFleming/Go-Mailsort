package rules

import (
	"testing"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name     string
		rules    []config.Rule
		email    *imapclient.Email
		wantNil  bool
		wantRule string
	}{
		{
			name: "matches from_contains",
			rules: []config.Rule{
				{Name: "rule1", FromContains: []string{"sender.com"}, MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@sender.com", Subject: "Subject"},
			wantNil:  false,
			wantRule: "rule1",
		},
		{
			name: "matches subject_any",
			rules: []config.Rule{
				{Name: "rule1", SubjectAny: []string{"Alert"}, MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@example.com", Subject: "Security Alert"},
			wantNil:  false,
			wantRule: "rule1",
		},
		{
			name: "no match - wrong sender",
			rules: []config.Rule{
				{Name: "rule1", FromContains: []string{"other.com"}, MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@example.com", Subject: "Subject"},
			wantNil:  true,
		},
		{
			name: "no match - wrong subject",
			rules: []config.Rule{
				{Name: "rule1", SubjectAny: []string{"Alert"}, MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@example.com", Subject: "Regular Email"},
			wantNil:  true,
		},
		{
			name:    "empty rules returns nil",
			rules:    []config.Rule{},
			email:    &imapclient.Email{From: "test@example.com", Subject: "Subject"},
			wantNil: true,
		},
		{
			name: "first matching rule wins (no chain)",
			rules: []config.Rule{
				{Name: "rule1", FromContains: []string{"test.com"}, MoveTo: "folder1"},
				{Name: "rule2", FromContains: []string{"test.com"}, MoveTo: "folder2"},
			},
			email:    &imapclient.Email{From: "test@test.com", Subject: "Subject"},
			wantNil:  false,
			wantRule: "rule1",
		},
		{
			name: "matches multiple from_contains",
			rules: []config.Rule{
				{Name: "rule1", FromContains: []string{"sender.com", "other.com"}, MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@other.com", Subject: "Subject"},
			wantNil:  false,
			wantRule: "rule1",
		},
		{
			name: "matches multiple subject_any",
			rules: []config.Rule{
				{Name: "rule1", SubjectAny: []string{"Alert", "Notification"}, MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@example.com", Subject: "Security Notification"},
			wantNil:  false,
			wantRule: "rule1",
		},
		{
			name: "empty rule criteria matches everything",
			rules: []config.Rule{
				{Name: "rule1", MoveTo: "folder1"},
			},
			email:    &imapclient.Email{From: "test@example.com", Subject: "Any Subject"},
			wantNil:  false,
			wantRule: "rule1",
		},
		{
			name: "chain rule - multiple matches",
			rules: []config.Rule{
				{Name: "rule1", FromContains: []string{"test.com"}, MoveTo: "folder1", Chain: true},
				{Name: "rule2", SubjectAny: []string{"Alert"}, MoveTo: "folder2", Chain: false},
			},
			email:    &imapclient.Email{From: "test@test.com", Subject: "Alert"},
			wantNil:  false,
			wantRule: "rule1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMatcher(tt.rules)
			got := m.Match(tt.email)

			if tt.wantNil {
				if len(got) > 0 {
					t.Errorf("Match() = %v rules, want nil", len(got))
				}
			} else {
				if len(got) == 0 {
					t.Error("Match() = nil, want rule")
				} else if tt.wantRule != "" && got[0].Name != tt.wantRule {
					t.Errorf("Match()[0] = %s, want %s", got[0].Name, tt.wantRule)
				}
			}
		})
	}
}

func TestMatch_Integration(t *testing.T) {
	rules := []config.Rule{
		{Name: "newsletter", FromContains: []string{"newsletter.com"}, MoveTo: "Newsletters"},
		{Name: "alerts", SubjectAny: []string{"Alert", "Warning"}, MoveTo: "Alerts"},
		{Name: "catch-all", MoveTo: "Inbox"},
	}

	tests := []struct {
		name    string
		email  *imapclient.Email
		want    string
	}{
		{
			name:   "newsletter from sender",
			email:  &imapclient.Email{From: "newsletter@newsletter.com", Subject: "Weekly Update"},
			want:   "newsletter",
		},
		{
			name:   "security alert by subject",
			email:  &imapclient.Email{From: "anyone@example.com", Subject: "Security Alert"},
			want:   "alerts",
		},
		{
			name:   "catch-all unmatched",
			email:  &imapclient.Email{From: "friend@gmail.com", Subject: "Hello"},
			want:   "catch-all",
		},
		{
			name:   "matches first rule only",
			email:  &imapclient.Email{From: "newsletter@newsletter.com", Subject: "Alert"},
			want:   "newsletter",
		},
	}

	m := NewMatcher(rules)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.Match(tt.email)
			if len(got) == 0 {
				t.Fatalf("Match() = nil, want %s", tt.want)
			}
			if got[0].Name != tt.want {
				t.Errorf("Match()[0] = %s, want %s", got[0].Name, tt.want)
			}
		})
	}
}

func TestMatch_Chaining(t *testing.T) {
	rules := []config.Rule{
		{Name: "rule1", FromContains: []string{"test.com"}, MoveTo: "folder1", Chain: true},
		{Name: "rule2", SubjectAny: []string{"Alert"}, MoveTo: "folder2", Chain: false},
		{Name: "rule3", MoveTo: "folder3", Chain: false},
	}

	email := &imapclient.Email{From: "test@test.com", Subject: "Alert"}

	m := NewMatcher(rules)
	got := m.Match(email)

	if len(got) != 2 {
		t.Fatalf("Match() returned %d rules, want 2", len(got))
	}

	if got[0].Name != "rule1" {
		t.Errorf("Match()[0] = %s, want rule1", got[0].Name)
	}
	if got[1].Name != "rule2" {
		t.Errorf("Match()[1] = %s, want rule2", got[1].Name)
	}
}
