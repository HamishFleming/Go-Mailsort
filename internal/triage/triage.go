package triage

import (
	"log"
	"strings"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
	"github.com/HamishFleming/Go-Mailsort/internal/rules"
)

type Message struct {
	Email       *imapclient.Email
	Rules       []*config.Rule
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
		matchedRules := matcher.Match(email)

		ruleNames := make([]string, len(matchedRules))
		for j, r := range matchedRules {
			ruleNames[j] = r.Name
		}
		log.Printf("[DEBUG] triage.NewTriage: email[%d] UID=%d, rules=%v", i, email.Uid, strings.Join(ruleNames, ", "))

		triages = append(triages, Message{
			Email: email,
			Rules: matchedRules,
		})

		if len(matchedRules) > 0 {
			matchedCount++
		}
	}

	log.Printf("[INFO] triage.NewTriage: matched %d/%d emails to rules", matchedCount, len(emails))

	return triages
}

func ruleNames(rs []*config.Rule) string {
	if len(rs) == 0 {
		return "<nil>"
	}
	names := make([]string, len(rs))
	for i, r := range rs {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}