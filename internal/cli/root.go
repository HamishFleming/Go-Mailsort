package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
		log.Printf("  mailbox=%s UID=%d from=%q subject=%q", email.Mailbox, email.Uid, email.From, email.Subject)
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

	emails, err := fetchRuleMailboxes(client, cfg)
	if err != nil {
		return err
	}

	matcher := rules.NewMatcherWithDefaultMailbox(cfg.Rules, defaultMailbox(cfg))

	log.Printf("matching %d emails against %d rules", len(emails), len(cfg.Rules))

	// Track statistics
	totalMatched := 0
	totalActions := 0
	ruleMatchCount := make(map[string]int)
	actionMatchCount := make(map[string]int)
	var matchedEmails []struct {
		uid     uint32
		subject string
		rules   []string
	}

	for _, email := range emails {
		matchedRules := matcher.Match(&email)
		score, plannedActions := planEmailActions(&email, matchedRules, cfg)
		if len(matchedRules) > 0 || len(plannedActions) > 0 {
			totalMatched++
			ruleNames := make([]string, 0, len(matchedRules))
			for _, rule := range matchedRules {
				ruleNames = append(ruleNames, rule.Name)
				ruleMatchCount[rule.Name]++
			}
			for _, action := range plannedActions {
				log.Printf("  mailbox=%s UID=%d score=%d subject=%q -> %s", email.Mailbox, email.Uid, score, email.Subject, action.description)
				actionMatchCount[action.summary]++
				totalActions++
			}
			matchedEmails = append(matchedEmails, struct {
				uid     uint32
				subject string
				rules   []string
			}{email.Uid, email.Subject, ruleNames})
		}
	}

	// Print summary
	log.Printf("")
	log.Printf("=== Summary ===")
	log.Printf("Total emails matched: %d/%d", totalMatched, len(emails))
	log.Printf("Total planned actions: %d", totalActions)
	log.Printf("")
	log.Printf("Actions:")
	logActionSummary(actionMatchCount)
	log.Printf("")
	log.Printf("Matches per rule:")
	for _, rule := range cfg.Rules {
		count := ruleMatchCount[rule.Name]
		if count > 0 {
			log.Printf("  %s: %d emails", rule.Name, count)
		}
	}

	return nil
}

func logActionSummary(actionMatchCount map[string]int) {
	if len(actionMatchCount) == 0 {
		log.Printf("  none")
		return
	}

	actions := make([]string, 0, len(actionMatchCount))
	for action := range actionMatchCount {
		actions = append(actions, action)
	}
	sort.Strings(actions)

	for _, action := range actions {
		log.Printf("  %s: %d emails", action, actionMatchCount[action])
	}
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

	emails, err := fetchRuleMailboxes(client, cfg)
	if err != nil {
		return err
	}

	matcher := rules.NewMatcherWithDefaultMailbox(cfg.Rules, defaultMailbox(cfg))

	log.Printf("applying rules to %d emails", len(emails))

	moved := 0
	batches := newActionBatches()
	for _, email := range emails {
		matchedRules := matcher.Match(&email)
		score, plannedActions := planEmailActions(&email, matchedRules, cfg)
		if len(matchedRules) > 0 || len(plannedActions) > 0 {
			for _, action := range plannedActions {
				log.Printf("  mailbox=%s UID=%d score=%d subject=%q -> %s", email.Mailbox, email.Uid, score, email.Subject, action.description)

				if !DryRun {
					stop := batches.add(&email, action)
					if stop {
						break
					}
				}
			}
			moved++
		}
	}

	if !DryRun {
		if err := batches.execute(client); err != nil {
			return err
		}
	}

	log.Printf("processed %d emails with matching rules", moved)
	return nil
}

type plannedAction struct {
	description string
	summary     string
	rule        *config.Rule
	autoMoveTo  string
}

func planEmailActions(email *imapclient.Email, matchedRules []*config.Rule, cfg *config.Config) (int, []plannedAction) {
	score := 0
	actions := make([]plannedAction, 0, len(matchedRules)+1)
	hasMoveOrDelete := false

	for _, rule := range matchedRules {
		score += rule.Score

		summary := describeRuleAction(rule)
		if summary == "no action" {
			continue
		}

		if rule.Delete || strings.TrimSpace(rule.MoveTo) != "" {
			hasMoveOrDelete = true
		}

		actions = append(actions, plannedAction{
			description: fmt.Sprintf("%s (rule: %s)", summary, rule.Name),
			summary:     summary,
			rule:        rule,
		})
	}

	if shouldAutoArchive(email, score, hasMoveOrDelete, cfg) {
		moveTo := autoArchiveMoveTo(cfg)
		summary := "auto archive to " + moveTo
		actions = append(actions, plannedAction{
			description: fmt.Sprintf("%s (score %d <= %d)", summary, score, cfg.AutoArchive.Threshold),
			summary:     summary,
			autoMoveTo:  moveTo,
		})
	}

	return score, actions
}

type actionBatches struct {
	copies  map[string][]uint32
	flags   map[string][]uint32
	reads   map[string][]uint32
	moves   map[string][]uint32
	deletes map[string][]uint32
}

func newActionBatches() *actionBatches {
	return &actionBatches{
		copies:  make(map[string][]uint32),
		flags:   make(map[string][]uint32),
		reads:   make(map[string][]uint32),
		moves:   make(map[string][]uint32),
		deletes: make(map[string][]uint32),
	}
}

func (b *actionBatches) add(email *imapclient.Email, action plannedAction) bool {
	if action.autoMoveTo != "" {
		b.addMove(email.Mailbox, action.autoMoveTo, email.Uid)
		return true
	}
	if action.rule == nil {
		return false
	}

	rule := action.rule
	if rule.Delete {
		b.addDelete(email.Mailbox, email.Uid)
		return true
	}

	if strings.TrimSpace(rule.CopyTo) != "" {
		b.addCopy(email.Mailbox, rule.CopyTo, email.Uid)
	}

	if rule.FlagImportant {
		b.addFlag(email.Mailbox, email.Uid)
	}

	if rule.MarkAsRead {
		b.addRead(email.Mailbox, email.Uid)
	}

	if strings.TrimSpace(rule.MoveTo) != "" {
		b.addMove(email.Mailbox, rule.MoveTo, email.Uid)
		return true
	}

	return false
}

func (b *actionBatches) addCopy(mailbox, folder string, uid uint32) {
	key := batchKey(mailbox, folder)
	b.copies[key] = append(b.copies[key], uid)
}

func (b *actionBatches) addFlag(mailbox string, uid uint32) {
	b.flags[mailbox] = append(b.flags[mailbox], uid)
}

func (b *actionBatches) addRead(mailbox string, uid uint32) {
	b.reads[mailbox] = append(b.reads[mailbox], uid)
}

func (b *actionBatches) addMove(mailbox, folder string, uid uint32) {
	key := batchKey(mailbox, folder)
	b.moves[key] = append(b.moves[key], uid)
}

func (b *actionBatches) addDelete(mailbox string, uid uint32) {
	b.deletes[mailbox] = append(b.deletes[mailbox], uid)
}

func (b *actionBatches) execute(client *imapclient.Client) error {
	for key, uids := range b.copies {
		mailbox, folder := splitBatchKey(key)
		log.Printf("batch copy %d messages from %s to %s", len(uids), mailbox, folder)
		if err := client.CopyMany(mailbox, uids, folder); err != nil {
			return fmt.Errorf("copy %s -> %s: %w", mailbox, folder, err)
		}
	}
	for mailbox, uids := range b.flags {
		log.Printf("batch flag important %d messages in %s", len(uids), mailbox)
		if err := client.FlagImportantMany(mailbox, uids); err != nil {
			return fmt.Errorf("flag important %s: %w", mailbox, err)
		}
	}
	for mailbox, uids := range b.reads {
		log.Printf("batch mark read %d messages in %s", len(uids), mailbox)
		if err := client.MarkAsReadMany(mailbox, uids); err != nil {
			return fmt.Errorf("mark read %s: %w", mailbox, err)
		}
	}
	for key, uids := range b.moves {
		mailbox, folder := splitBatchKey(key)
		log.Printf("batch move %d messages from %s to %s", len(uids), mailbox, folder)
		if err := client.MoveMany(mailbox, uids, folder); err != nil {
			return fmt.Errorf("move %s -> %s: %w", mailbox, folder, err)
		}
	}
	for mailbox, uids := range b.deletes {
		log.Printf("batch delete %d messages from %s", len(uids), mailbox)
		if err := client.DeleteMany(mailbox, uids); err != nil {
			return fmt.Errorf("delete %s: %w", mailbox, err)
		}
	}

	return nil
}

func batchKey(mailbox, folder string) string {
	return mailbox + "\x00" + folder
}

func splitBatchKey(key string) (string, string) {
	parts := strings.SplitN(key, "\x00", 2)
	if len(parts) != 2 {
		return key, ""
	}
	return parts[0], parts[1]
}

func shouldAutoArchive(email *imapclient.Email, score int, hasMoveOrDelete bool, cfg *config.Config) bool {
	if !cfg.AutoArchive.Enabled || hasMoveOrDelete {
		return false
	}
	if email.Flagged {
		return false
	}
	if time.Since(email.Date) < 24*time.Hour {
		return false
	}
	if cfg.AutoArchive.DateBefore != "" {
		before, err := parseAutoArchiveDate(cfg.AutoArchive.DateBefore)
		if err != nil {
			log.Printf("[WARN] invalid auto_archive.date_before format: %s", cfg.AutoArchive.DateBefore)
			return false
		}
		if email.Date.After(before) {
			return false
		}
	}

	source := strings.TrimSpace(cfg.AutoArchive.Folder)
	if source == "" {
		source = defaultMailbox(cfg)
	}

	emailMailbox := strings.TrimSpace(email.Mailbox)
	if emailMailbox == "" {
		emailMailbox = "INBOX"
	}

	return strings.EqualFold(source, emailMailbox) && score <= cfg.AutoArchive.Threshold
}

func parseAutoArchiveDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "-") && strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(strings.TrimPrefix(s, "-"), "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days <= 0 {
			return time.Time{}, fmt.Errorf("invalid relative date: %s", s)
		}
		return time.Now().AddDate(0, 0, -days), nil
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid date: %s", s)
}

func autoArchiveMoveTo(cfg *config.Config) string {
	if moveTo := strings.TrimSpace(cfg.AutoArchive.MoveTo); moveTo != "" {
		return moveTo
	}
	return "Archive"
}

func fetchRuleMailboxes(client *imapclient.Client, cfg *config.Config) ([]imapclient.Email, error) {
	var emails []imapclient.Email

	for _, mailbox := range sourceMailboxes(cfg) {
		fetched, err := client.Fetch(mailbox)
		if err != nil {
			return nil, err
		}
		emails = append(emails, fetched...)
	}

	return emails, nil
}

func sourceMailboxes(cfg *config.Config) []string {
	seen := make(map[string]struct{})
	var mailboxes []string

	add := func(mailbox string) {
		mailbox = strings.TrimSpace(mailbox)
		if mailbox == "" {
			mailbox = "INBOX"
		}
		key := strings.ToLower(mailbox)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		mailboxes = append(mailboxes, mailbox)
	}

	add(defaultMailbox(cfg))
	for _, rule := range cfg.Rules {
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		if rule.Folder != "" {
			add(rule.Folder)
		}
	}

	return mailboxes
}

func defaultMailbox(cfg *config.Config) string {
	mailbox := strings.TrimSpace(cfg.Mailbox)
	if mailbox == "" {
		return "INBOX"
	}
	return mailbox
}

func describeRuleAction(rule *config.Rule) string {
	if rule.Delete {
		return "delete"
	}

	var actions []string
	if strings.TrimSpace(rule.CopyTo) != "" {
		actions = append(actions, "copy to "+rule.CopyTo)
	}
	if rule.FlagImportant {
		actions = append(actions, "flag important")
	}
	if strings.TrimSpace(rule.MoveTo) != "" {
		actions = append(actions, "move to "+rule.MoveTo)
	}
	if rule.MarkAsRead {
		actions = append(actions, "mark as read")
	}
	if len(actions) == 0 {
		return "no action"
	}
	return strings.Join(actions, ", ")
}

func Init(cfg *config.Config) error {
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

	mailboxes := requiredMailboxes(cfg)
	if len(mailboxes) == 0 {
		log.Printf("no required mailboxes configured")
		return nil
	}

	log.Printf("checking %d required IMAP mailboxes", len(mailboxes))
	created, err := client.EnsureMailboxes(mailboxes)
	if err != nil {
		return err
	}

	if len(created) == 0 {
		log.Printf("all required IMAP mailboxes already exist")
		return nil
	}

	log.Printf("created %d IMAP mailboxes", len(created))
	return nil
}

func requiredMailboxes(cfg *config.Config) []string {
	seen := make(map[string]struct{})
	var mailboxes []string

	add := func(mailbox string) {
		mailbox = strings.TrimSpace(mailbox)
		if mailbox == "" {
			return
		}
		if _, ok := seen[mailbox]; ok {
			return
		}
		seen[mailbox] = struct{}{}
		mailboxes = append(mailboxes, mailbox)
	}

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}
	add(mailbox)
	if cfg.AutoArchive.Enabled {
		add(cfg.AutoArchive.Folder)
		add(autoArchiveMoveTo(cfg))
	}

	for _, rule := range cfg.Rules {
		if rule.Enabled != nil && !*rule.Enabled {
			continue
		}
		add(rule.Folder)
		add(rule.MoveTo)
		add(rule.CopyTo)
	}

	return mailboxes
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
		if rule.Folder != "" {
			log.Printf("    folder: %s", rule.Folder)
		}
		if rule.Score != 0 {
			log.Printf("    score: %d", rule.Score)
		}
		if rule.Unread != nil {
			log.Printf("    unread: %t", *rule.Unread)
		}
		if rule.OlderThan != nil {
			log.Printf("    older_than: %s", *rule.OlderThan)
		}
		if rule.NewerThan != nil {
			log.Printf("    newer_than: %s", *rule.NewerThan)
		}
		log.Printf("    move_to: %s", rule.MoveTo)
		if rule.CopyTo != "" {
			log.Printf("    copy_to: %s", rule.CopyTo)
		}
		log.Printf("    delete: %t", rule.Delete)
		log.Printf("    flag_important: %t", rule.FlagImportant)
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
	case "enable":
		return setRuleEnabled(cfg, rulesDir, args[1:], true)
	case "disable":
		return setRuleEnabled(cfg, rulesDir, args[1:], false)
	default:
		return fmt.Errorf("unknown rules subcommand: %s\nValid subcommands: list, add, remove, update, enable, disable", subcommand)
	}
}

func setRuleEnabled(cfg *config.Config, rulesDir string, args []string, enabled bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mailsort rules %s <name>", map[bool]string{true: "enable", false: "disable"}[enabled])
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

	cfg.Rules[index].Enabled = &enabled
	log.Printf("rule '%s' %s", name, map[bool]string{true: "enabled", false: "disabled"}[enabled])

	// Update the rule file in the rules directory
	filename := filepath.Join(rulesDir, fmt.Sprintf("%d-%s.yaml", cfg.Rules[index].Priority, name))
	data, err := yaml.Marshal([]config.Rule{cfg.Rules[index]})
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}
