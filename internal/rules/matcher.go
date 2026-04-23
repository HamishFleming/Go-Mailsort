package rules

import (
	"strings"

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

	return true
}