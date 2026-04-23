package cli

import (
	"reflect"
	"testing"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
)

func TestRequiredMailboxes(t *testing.T) {
	disabled := false

	cfg := &config.Config{
		Mailbox: "INBOX",
		Rules: []config.Rule{
			{Name: "github", MoveTo: "GitHub"},
			{Name: "github-duplicate", MoveTo: "GitHub"},
			{Name: "disabled", MoveTo: "Disabled", Enabled: &disabled},
			{Name: "empty"},
			{Name: "trimmed", MoveTo: " Newsletters "},
		},
	}

	got := requiredMailboxes(cfg)
	want := []string{"INBOX", "GitHub", "Newsletters"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requiredMailboxes() = %#v, want %#v", got, want)
	}
}

func TestRequiredMailboxesDefaultsToInbox(t *testing.T) {
	got := requiredMailboxes(&config.Config{})
	want := []string{"INBOX"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requiredMailboxes() = %#v, want %#v", got, want)
	}
}
