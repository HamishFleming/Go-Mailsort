package triage

import (
	"log"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
	"github.com/HamishFleming/Go-Mailsort/internal/rules"
)

type Message struct {
	Email      *imapclient.Email
	Rule       *config.Rule
}

type Triage struct {
	messages []Message
}

func NewTriage(emails []imapclient.Email, matcher *rules.Matcher) []Message {
	log.Printf("[DEBUG] triage.NewTriage: processing %d emails", len(emails))

	triages := make([]Message, 0, len(emails))
	matchedCount := 0

	for i := range emails {
		email := &emails[i]
		rule := matcher.Match(email)

		log.Printf("[DEBUG] triage.NewTriage: email[%d] UID=%d, rule=%v", i, email.Uid, ruleName(rule))

		triages = append(triages, Message{
			Email: email,
			Rule:  rule,
		})

		if rule != nil {
			matchedCount++
		}
	}

	log.Printf("[INFO] triage.NewTriage: matched %d/%d emails to rules", matchedCount, len(emails))

	return triages
}

func ruleName(r *config.Rule) string {
	if r == nil {
		return "<nil>"
	}
	return r.Name
}