package config

import (
	"fmt"
	"log"
	"os"

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

type Config struct {
	Mailbox string `yaml:"mailbox"`
	Rules   []Rule  `yaml:"rules"`
}

type Rule struct {
	Name         string   `yaml:"name"`
	FromContains []string `yaml:"from_contains"`
	SubjectAny   []string `yaml:"subject_any"`
	BodyAny      []string `yaml:"body_any"`
	MoveTo       string   `yaml:"move_to"`
	MarkAsRead   bool     `yaml:"mark_as_read"`
}