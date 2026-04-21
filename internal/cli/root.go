package cli

import (
	"log"

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

func Rules(cfg *config.Config, args []string) error {
	log.Printf("rules not yet implemented")
	return nil
}