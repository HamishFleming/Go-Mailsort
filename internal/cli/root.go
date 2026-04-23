package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient/yahoo"
	"github.com/HamishFleming/Go-Mailsort/internal/rules"
	"gopkg.in/yaml.v3"
)

var (
	Verbose bool
	DryRun  bool
)

func Scan(cfg *config.Config) error {
	provider, err := yahoo.NewProvider()
	if err != nil {
		return err
	}

	imapCfg := &imapclient.Config{Mailbox: cfg.Mailbox}
	client, err := provider.Connect(imapCfg)
	if err != nil {
		return err
	}
	defer client.Close()

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	emails, err := client.FetchUnread(mailbox)
	if err != nil {
		return err
	}

	log.Printf("found %d unread emails", len(emails))

	for _, email := range emails {
		log.Printf("  UID=%d from=%q subject=%q", email.Uid, email.From, email.Subject)
	}

	return nil
}

func Preview(cfg *config.Config) error {
	provider, err := yahoo.NewProvider()
	if err != nil {
		return err
	}

	imapCfg := &imapclient.Config{Mailbox: cfg.Mailbox}
	client, err := provider.Connect(imapCfg)
	if err != nil {
		return err
	}
	defer client.Close()

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	emails, err := client.FetchUnread(mailbox)
	if err != nil {
		return err
	}

	matcher := rules.NewMatcher(cfg.Rules)

	log.Printf("matching %d emails against %d rules", len(emails), len(cfg.Rules))

	for _, email := range emails {
		matchedRules := matcher.Match(&email)
		if len(matchedRules) > 0 {
			for _, rule := range matchedRules {
				log.Printf("  UID=%d subject=%q -> %s (rule: %s)", email.Uid, email.Subject, rule.MoveTo, rule.Name)
			}
		}
	}

	return nil
}

func Apply(cfg *config.Config) error {
	if DryRun {
		log.Printf("[DRY-RUN] applying rules in dry-run mode")
	}

	provider, err := yahoo.NewProvider()
	if err != nil {
		return err
	}

	imapCfg := &imapclient.Config{Mailbox: cfg.Mailbox}
	client, err := provider.Connect(imapCfg)
	if err != nil {
		return err
	}
	defer client.Close()

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	emails, err := client.FetchUnread(mailbox)
	if err != nil {
		return err
	}

	matcher := rules.NewMatcher(cfg.Rules)

	log.Printf("applying rules to %d emails", len(emails))

	moved := 0
	for _, email := range emails {
		matchedRules := matcher.Match(&email)
		if len(matchedRules) > 0 {
			for _, rule := range matchedRules {
				log.Printf("  UID=%d subject=%q -> %s (rule: %s)", email.Uid, email.Subject, rule.MoveTo, rule.Name)

				if !DryRun {
					if err := client.Move(email.Uid, rule.MoveTo); err != nil {
						log.Printf("ERROR: move failed: %v", err)
						continue
					}

					if rule.MarkAsRead {
						if err := client.MarkAsRead(email.Uid); err != nil {
							log.Printf("WARN: mark as read failed: %v", err)
						}
					}
				}
			}
			moved++
		}
	}

	log.Printf("processed %d emails with matching rules", moved)
	return nil
}

func listRules(cfg *config.Config) error {
	if len(cfg.Rules) == 0 {
		log.Printf("no rules configured")
		return nil
	}

	for i, rule := range cfg.Rules {
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		log.Printf("%d. %s (priority: %d, enabled: %t):", i+1, rule.Name, rule.Priority, enabled)
		if len(rule.FromContains) > 0 {
			log.Printf("    from_contains: %s", strings.Join(rule.FromContains, ", "))
		}
		if len(rule.SubjectAny) > 0 {
			log.Printf("    subject_any: %s", strings.Join(rule.SubjectAny, ", "))
		}
		if len(rule.BodyAny) > 0 {
			log.Printf("    body_any: %s", strings.Join(rule.BodyAny, ", "))
		}
		log.Printf("    move_to: %s", rule.MoveTo)
		log.Printf("    mark_as_read: %t", rule.MarkAsRead)
		log.Printf("    chain: %t", rule.Chain)
		log.Printf("")
	}

	return nil
}

func addRule(cfg *config.Config, rulesDir string, args []string) error {
	if len(args) < 8 {
		return fmt.Errorf("usage: mailsort rules add <priority> <name> <enabled> <from_contains> <subject_any> <body_any> <move_to> <mark_as_read> [chain]")
	}

	priority, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	name := args[1]
	enabled, err := parseBool(args[2])
	if err != nil {
		return fmt.Errorf("invalid enabled value: %w", err)
	}
	fromContains := splitCsvFixed(args[3])
	subjectAny := splitCsvFixed(args[4])
	bodyAny := splitCsvFixed(args[5])
	moveTo := args[6]
	markAsRead, err := parseBool(args[7])
	if err != nil {
		return fmt.Errorf("invalid mark_as_read value: %w", err)
	}

	chain := false
	if len(args) > 8 {
		chain, err = parseBool(args[8])
		if err != nil {
			return fmt.Errorf("invalid chain value: %w", err)
		}
	}

	rule := config.Rule{
		Name:         name,
		Enabled:      &enabled,
		Priority:     priority,
		FromContains: fromContains,
		SubjectAny:   subjectAny,
		BodyAny:      bodyAny,
		MoveTo:       moveTo,
		MarkAsRead:   markAsRead,
		Chain:        chain,
	}

	cfg.Rules = append(cfg.Rules, rule)
	log.Printf("rule '%s' added with priority %d, enabled=%t", name, priority, enabled)

	// Save to a file in the rules directory
	filename := filepath.Join(rulesDir, fmt.Sprintf("%d-%s.yaml", priority, name))
	data, err := yaml.Marshal([]config.Rule{rule})
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}

func removeRule(cfg *config.Config, rulesDir string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mailsort rules remove <name>")
	}

	name := args[0]
	index := -1
	for i, rule := range cfg.Rules {
		if rule.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("rule '%s' not found", name)
	}

	cfg.Rules = append(cfg.Rules[:index], cfg.Rules[index+1:]...)
	log.Printf("rule '%s' removed", name)

	// Remove the rule file from the rules directory
	// Find and remove the file containing this rule
	files, err := os.ReadDir(rulesDir)
	if err != nil {
		return fmt.Errorf("read rules dir: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		path := filepath.Join(rulesDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var rules []config.Rule
		if err := yaml.Unmarshal(data, &rules); err != nil {
			continue
		}

		for _, r := range rules {
			if r.Name == name {
				os.Remove(path)
				return nil
			}
		}
	}

	return nil
}

func updateRule(cfg *config.Config, rulesDir string, args []string) error {
	if len(args) < 8 {
		return fmt.Errorf("usage: mailsort rules update <name> <priority> <enabled> <from_contains> <subject_any> <body_any> <move_to> <mark_as_read> [chain]")
	}

	name := args[0]
	priority, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

	enabled, err := parseBool(args[2])
	if err != nil {
		return fmt.Errorf("invalid enabled value: %w", err)
	}

	fromContains := splitCsvFixed(args[3])
	subjectAny := splitCsvFixed(args[4])
	bodyAny := splitCsvFixed(args[5])
	moveTo := args[6]
	markAsRead, err := parseBool(args[7])
	if err != nil {
		return fmt.Errorf("invalid mark_as_read value: %w", err)
	}

	chain := false
	if len(args) > 8 {
		chain, err = parseBool(args[8])
		if err != nil {
			return fmt.Errorf("invalid chain value: %w", err)
		}
	}

	index := -1
	for i, rule := range cfg.Rules {
		if rule.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("rule '%s' not found", name)
	}

	cfg.Rules[index] = config.Rule{
		Name:         name,
		Enabled:      &enabled,
		Priority:     priority,
		FromContains: fromContains,
		SubjectAny:   subjectAny,
		BodyAny:      bodyAny,
		MoveTo:       moveTo,
		MarkAsRead:   markAsRead,
		Chain:        chain,
	}

	log.Printf("rule '%s' updated with priority %d, enabled=%t", name, priority, enabled)

	// Update the rule file in the rules directory
	filename := filepath.Join(rulesDir, fmt.Sprintf("%d-%s.yaml", priority, name))
	data, err := yaml.Marshal([]config.Rule{cfg.Rules[index]})
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}

func splitCsv(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitCsvFixed(s string) []string {
	if s == "" {
		return []string{}
	}
	// Handle comma-separated values
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}

func Rules(cfg *config.Config, rulesDir string, args []string) error {
	if len(args) == 0 {
		return listRules(cfg)
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		return listRules(cfg)
	case "add":
		return addRule(cfg, rulesDir, args[1:])
	case "remove":
		return removeRule(cfg, rulesDir, args[1:])
	case "update":
		return updateRule(cfg, rulesDir, args[1:])
	default:
		return fmt.Errorf("unknown rules subcommand: %s\nValid subcommands: list, add, remove, update", subcommand)
	}
}