package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	log.Printf("[DEBUG] config.Load: reading %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[ERROR] config.Load: read: %v", err)
		return nil, fmt.Errorf("read: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("[ERROR] config.Load: parse: %v", err)
		return nil, fmt.Errorf("parse: %w", err)
	}

	log.Printf("[INFO] config loaded: mailbox=%s", cfg.Mailbox)
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	log.Printf("[DEBUG] config.Save: writing %s", path)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Printf("[ERROR] config.Save: marshal: %v", err)
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[ERROR] config.Save: write: %v", err)
		return fmt.Errorf("write: %w", err)
	}

	log.Printf("[INFO] config saved: mailbox=%s", cfg.Mailbox)
	return nil
}

func LoadRulesFromDir(dir string) ([]Rule, error) {
	log.Printf("[DEBUG] config.LoadRulesFromDir: reading directory %s", dir)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var allRules []Rule
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := filepath.Ext(file.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[ERROR] config.LoadRulesFromDir: read %s: %v", path, err)
			continue
		}

		var rules []Rule
		if err := yaml.Unmarshal(data, &rules); err != nil {
			log.Printf("[ERROR] config.LoadRulesFromDir: parse %s: %v", path, err)
			continue
		}

		allRules = append(allRules, rules...)
	}

	sort.SliceStable(allRules, func(i, j int) bool {
		return allRules[i].Priority < allRules[j].Priority
	})

	log.Printf("[INFO] loaded %d rules from %s", len(allRules), dir)
	return allRules, nil
}

type Config struct {
	Mailbox     string            `yaml:"mailbox"`
	RulesDir    string            `yaml:"rules_dir"`
	AutoArchive AutoArchiveConfig `yaml:"auto_archive"`
	Rules       []Rule            `yaml:"rules"`
}

func LoadMainConfig(path string) (*Config, error) {
	log.Printf("[DEBUG] config.LoadMainConfig: reading %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[ERROR] config.LoadMainConfig: read: %v", err)
		return nil, fmt.Errorf("read: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("[ERROR] config.LoadMainConfig: parse: %v", err)
		return nil, fmt.Errorf("parse: %w", err)
	}

	log.Printf("[INFO] main config loaded: mailbox=%s, rules_dir=%s", cfg.Mailbox, cfg.RulesDir)
	return &cfg, nil
}

type Rule struct {
	Name           string   `yaml:"name"`
	Enabled        *bool    `yaml:"enabled,omitempty"`
	Priority       int      `yaml:"priority"`
	Score          int      `yaml:"score,omitempty"`
	Folder         string   `yaml:"folder,omitempty"`
	FromContains   []string `yaml:"from_contains"`
	SubjectAny     []string `yaml:"subject_any"`
	BodyAny        []string `yaml:"body_any"`
	DateAfter      *string  `yaml:"date_after,omitempty"`  // RFC3339 format: "2024-01-15"
	DateBefore     *string  `yaml:"date_before,omitempty"` // RFC3339 format: "2024-12-31"
	OlderThan      *string  `yaml:"older_than,omitempty"`  // duration format: "30d", "12h"
	NewerThan      *string  `yaml:"newer_than,omitempty"`  // duration format: "7d", "24h"
	Unread         *bool    `yaml:"unread,omitempty"`
	HasAttachments *bool    `yaml:"has_attachments,omitempty"`
	MinSize        *uint32  `yaml:"min_size,omitempty"` // in bytes
	MaxSize        *uint32  `yaml:"max_size,omitempty"` // in bytes
	MoveTo         string   `yaml:"move_to"`
	CopyTo         string   `yaml:"copy_to,omitempty"`
	Delete         bool     `yaml:"delete,omitempty"`
	FlagImportant  bool     `yaml:"flag_important,omitempty"`
	MarkAsRead     bool     `yaml:"mark_as_read"`
	Chain          bool     `yaml:"chain"`
}

type AutoArchiveConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Threshold  int    `yaml:"threshold"`
	MoveTo     string `yaml:"move_to"`
	Folder     string `yaml:"folder,omitempty"`
	DateBefore string `yaml:"date_before,omitempty"`
}
