package config

import (
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		filename  string
		wantErr   bool
		wantMail  string
		wantRules int
	}{
		{
			name:       "valid config",
			filename:  "testdata/valid.yaml",
			wantErr:   false,
			wantMail:  "INBOX",
			wantRules: 2,
		},
		{
			name:     "file not found",
			filename: "testdata/nonexistent.yaml",
			wantErr:  true,
		},
		{
			name:     "invalid yaml",
			filename: "testdata/invalid.yaml",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if cfg.Mailbox != tt.wantMail {
				t.Errorf("Load().Mailbox = %q, want %q", cfg.Mailbox, tt.wantMail)
			}

			if len(cfg.Rules) != tt.wantRules {
				t.Errorf("Load().Rules = %d, want %d", len(cfg.Rules), tt.wantRules)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid.yaml"))
	if err == nil {
		t.Error("Load() expected error for invalid YAML")
	}
}