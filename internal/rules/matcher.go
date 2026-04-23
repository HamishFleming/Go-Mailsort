package rules

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

type Matcher struct {
	rules []config.Rule
}

func NewMatcher(rules []config.Rule) *Matcher {
	return &Matcher{rules: rules}
}

func (m *Matcher) Match(email *imapclient.Email) []*config.Rule {
	var matchedRules []*config.Rule
	for i := range m.rules {
		if m.matchRule(&m.rules[i], email) {
			matchedRules = append(matchedRules, &m.rules[i])
			if !m.rules[i].Chain {
				break
			}
		}
	}
	return matchedRules
}

func (m *Matcher) matchRule(rule *config.Rule, email *imapclient.Email) bool {
	if len(rule.FromContains) > 0 {
		matched := false
		for _, f := range rule.FromContains {
			if strings.Contains(email.From, f) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(rule.SubjectAny) > 0 {
		matched := false
		for _, s := range rule.SubjectAny {
			if strings.Contains(email.Subject, s) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Date filtering
	if rule.DateAfter != nil {
		afterDate, err := parseDate(*rule.DateAfter)
		if err != nil {
			log.Printf("[WARN] invalid date_after format: %s", *rule.DateAfter)
		} else if email.Date.Before(afterDate) {
			return false
		}
	}

	if rule.DateBefore != nil {
		beforeDate, err := parseDate(*rule.DateBefore)
		if err != nil {
			log.Printf("[WARN] invalid date_before format: %s", *rule.DateBefore)
		} else if email.Date.After(beforeDate) {
			return false
		}
	}

	// Attachment filtering
	if rule.HasAttachments != nil {
		if email.HasAttachments != *rule.HasAttachments {
			return false
		}
	}

	// Size filtering
	if rule.MinSize != nil {
		if email.Size < *rule.MinSize {
			return false
		}
	}

	if rule.MaxSize != nil {
		if email.Size > *rule.MaxSize {
			return false
		}
	}

	return true
}

func parseDate(s string) (time.Time, error) {
	// Try RFC3339 with time first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try date-only format (add time 00:00:00)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}