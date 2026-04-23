package rules

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapclient"
)

type Matcher struct {
	rules          []config.Rule
	defaultMailbox string
}

func NewMatcher(rules []config.Rule) *Matcher {
	return &Matcher{rules: rules}
}

func NewMatcherWithDefaultMailbox(rules []config.Rule, defaultMailbox string) *Matcher {
	return &Matcher{rules: rules, defaultMailbox: strings.TrimSpace(defaultMailbox)}
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
	// Check if rule is enabled (default true if not specified)
	if rule.Enabled != nil && !*rule.Enabled {
		return false
	}

	folder := strings.TrimSpace(rule.Folder)
	if folder == "" {
		folder = m.defaultMailbox
	}
	if folder != "" {
		emailMailbox := strings.TrimSpace(email.Mailbox)
		if emailMailbox == "" {
			emailMailbox = "INBOX"
		}
		if !strings.EqualFold(folder, emailMailbox) {
			return false
		}
	}

	if len(rule.FromContains) > 0 {
		matched := false
		for _, f := range rule.FromContains {
			if containsFold(email.From, f) {
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
			if containsFold(email.Subject, s) {
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

	if rule.OlderThan != nil {
		olderThan, err := parseAge(*rule.OlderThan)
		if err != nil {
			log.Printf("[WARN] invalid older_than format: %s", *rule.OlderThan)
		} else if time.Since(email.Date) < olderThan {
			return false
		}
	}

	if rule.NewerThan != nil {
		newerThan, err := parseAge(*rule.NewerThan)
		if err != nil {
			log.Printf("[WARN] invalid newer_than format: %s", *rule.NewerThan)
		} else if time.Since(email.Date) > newerThan {
			return false
		}
	}

	if rule.Unread != nil {
		if email.Unread != *rule.Unread {
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

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func parseDate(s string) (time.Time, error) {
	// Try relative date format (e.g., "-30d" for 30 days ago)
	if strings.HasPrefix(s, "-") && strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(strings.TrimPrefix(s, "-"), "d")
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			return time.Now().AddDate(0, 0, -days), nil
		}
	}

	// Try RFC3339 with time
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try date-only format (add time 00:00:00)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}

func parseAge(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("invalid day duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return time.ParseDuration(s)
}
