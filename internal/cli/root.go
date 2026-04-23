package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient/yahoo"
	"github.com/HamishFleming/Go-Mailsort/internal/rules"
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
		rule := matcher.Match(&email)
		if rule != nil {
			log.Printf("  UID=%d match=%q -> %s", email.Uid, rule.Name, rule.MoveTo)
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
		rule := matcher.Match(&email)
		if rule != nil {
			log.Printf("would move UID=%d to %s", email.Uid, rule.MoveTo)

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

			moved++
		}
	}

	log.Printf("moved %d emails", moved)
	return nil
}

func listRules(cfg *config.Config) error {
	if len(cfg.Rules) == 0 {
		log.Printf("no rules configured")
		return nil
	}

	for i, rule := range cfg.Rules {
		log.Printf("%d. %s:", i+1, rule.Name)
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
		log.Printf("")
	}

	return nil
}

func addRule(cfg *config.Config, args []string) error {
	if len(args) < 6 {
		return fmt.Errorf("usage: mailsort rules add <name> <from_contains> <subject_any> <body_any> <move_to> <mark_as_read>")
	}

	name := args[0]
	fromContains := splitCsvFixed(args[1])
	subjectAny := splitCsvFixed(args[2])
	bodyAny := splitCsvFixed(args[3])
	moveTo := args[4]
	markAsRead, err := parseBool(args[5])
	if err != nil {
		return fmt.Errorf("invalid mark_as_read value: %w", err)
	}

	rule := config.Rule{
		Name:         name,
		FromContains: fromContains,
		SubjectAny:   subjectAny,
		BodyAny:      bodyAny,
		MoveTo:       moveTo,
		MarkAsRead:   markAsRead,
	}

	cfg.Rules = append(cfg.Rules, rule)
	log.Printf("rule '%s' added", name)
	return config.Save(".mailsort.yaml", cfg)
}

func removeRule(cfg *config.Config, args []string) error {
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
	return config.Save(".mailsort.yaml", cfg)
}

func updateRule(cfg *config.Config, args []string) error {
	if len(args) < 6 {
		return fmt.Errorf("usage: mailsort rules update <name> <from_contains> <subject_any> <body_any> <move_to> <mark_as_read>")
	}

	name := args[0]
	fromContains := splitCsvFixed(args[1])
	subjectAny := splitCsvFixed(args[2])
	bodyAny := splitCsvFixed(args[3])
	moveTo := args[4]
	markAsRead, err := parseBool(args[5])
	if err != nil {
		return fmt.Errorf("invalid mark_as_read value: %w", err)
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
		FromContains: fromContains,
		SubjectAny:   subjectAny,
		BodyAny:      bodyAny,
		MoveTo:       moveTo,
		MarkAsRead:   markAsRead,
	}

	log.Printf("rule '%s' updated", name)
	return config.Save(".mailsort.yaml", cfg)
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

func Rules(cfg *config.Config, args []string) error {
	if len(args) == 0 {
		return listRules(cfg)
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		return listRules(cfg)
	case "add":
		return addRule(cfg, args[1:])
	case "remove":
		return removeRule(cfg, args[1:])
	case "update":
		return updateRule(cfg, args[1:])
	default:
		return fmt.Errorf("unknown rules subcommand: %s\nValid subcommands: list, add, remove, update", subcommand)
	}
}