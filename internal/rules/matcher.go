package rules

import (
	"strings"

	"github.com/yourname/mailsort/internal/config"
	"github.com/yourname/mailsort/internal/imapclient"
)

type Matcher struct {
	rules []config.Rule
}

func NewMatcher(rules []config.Rule) *Matcher {
	return &Matcher{rules: rules}
}

func (m *Matcher) Match(email *imapclient.Email) *config.Rule {
	for i := range m.rules {
		if m.matchRule(&m.rules[i], email) {
			return &m.rules[i]
		}
	}
	return nil
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

	return true
}