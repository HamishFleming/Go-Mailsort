package imapclient

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Email struct {
	Uid            uint32
	Mailbox        string
	From           string
	Subject        string
	Body           string
	Date           time.Time
	Size           uint32
	Unread         bool
	Flagged        bool
	HasAttachments bool
}

type Config struct {
	Host    string
	Port    int
	User    string
	Pass    string
	UseTLS  bool
	Mailbox string
}

type Provider interface {
	Connect(cfg *Config) (*Client, error)
}

type Client struct {
	cfg  *Config
	conn *client.Client
}

const maxUIDBatchSize = 5

func Connect(cfg *Config) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[INFO] connecting to %s", addr)

	var conn *client.Client
	var err error

	if cfg.UseTLS {
		conn, err = client.DialTLS(addr, nil)
	} else {
		conn, err = client.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	log.Printf("[DEBUG] connected")

	if err := conn.Login(cfg.User, cfg.Pass); err != nil {
		conn.Logout()
		return nil, fmt.Errorf("login: %w", err)
	}
	log.Printf("[DEBUG] authenticated")

	return &Client{cfg: cfg, conn: conn}, nil
}

func (c *Client) Close() error {
	if err := c.conn.Logout(); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

func (c *Client) FetchUnread(mailbox string) ([]Email, error) {
	emails, err := c.Fetch(mailbox)
	if err != nil {
		return nil, err
	}

	unread := emails[:0]
	for _, email := range emails {
		if email.Unread {
			unread = append(unread, email)
		}
	}

	return unread, nil
}

func (c *Client) Fetch(mailbox string) ([]Email, error) {
	mboxInfo, err := c.conn.Select(mailbox, false)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	log.Printf("[DEBUG] selected %s, messages=%d", mailbox, mboxInfo.Messages)

	if mboxInfo.Messages == 0 {
		return []Email{}, nil
	}

	seqSet := &imap.SeqSet{}
	seqSet.AddRange(1, mboxInfo.Messages)

	ch := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.conn.Fetch(seqSet, []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchUid,
			imap.FetchFlags,
			imap.FetchBodyStructure,
			imap.FetchRFC822Size,
		}, ch)
	}()

	var emails []Email
	for msg := range ch {
		if msg == nil {
			break
		}

		from := ""
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}

		hasAttachments := false
		if msg.BodyStructure != nil {
			hasAttachments = checkForAttachments(msg.BodyStructure)
		}

		emails = append(emails, Email{
			Uid:            msg.Uid,
			Mailbox:        mailbox,
			From:           from,
			Subject:        msg.Envelope.Subject,
			Date:           msg.Envelope.Date,
			Size:           msg.Size,
			Unread:         !hasFlag(msg.Flags, imap.SeenFlag),
			Flagged:        hasFlag(msg.Flags, imap.FlaggedFlag),
			HasAttachments: hasAttachments,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	log.Printf("[DEBUG] fetched %d emails", len(emails))
	return emails, nil
}

func (c *Client) EnsureMailboxes(mailboxes []string) ([]string, error) {
	existing, err := c.mailboxSet()
	if err != nil {
		return nil, err
	}

	var created []string
	for _, mailbox := range mailboxes {
		if mailbox == "" {
			continue
		}

		if mailboxExists(existing, mailbox) {
			log.Printf("[INFO] mailbox exists: %s", mailbox)
			continue
		}

		if strings.EqualFold(mailbox, "INBOX") {
			log.Printf("[INFO] mailbox exists: %s", mailbox)
			continue
		}

		if err := c.conn.Create(mailbox); err != nil {
			return created, fmt.Errorf("create mailbox %q: %w", mailbox, err)
		}
		existing[mailbox] = struct{}{}
		created = append(created, mailbox)
		log.Printf("[INFO] created mailbox: %s", mailbox)
	}

	return created, nil
}

func (c *Client) mailboxSet() (map[string]struct{}, error) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.conn.List("", "*", mailboxes)
	}()

	existing := make(map[string]struct{})
	for mailbox := range mailboxes {
		if mailbox == nil {
			continue
		}
		existing[mailbox.Name] = struct{}{}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("list mailboxes: %w", err)
	}

	return existing, nil
}

func mailboxExists(existing map[string]struct{}, mailbox string) bool {
	if _, ok := existing[mailbox]; ok {
		return true
	}

	if strings.EqualFold(mailbox, "INBOX") {
		for name := range existing {
			if strings.EqualFold(name, mailbox) {
				return true
			}
		}
	}

	return false
}

func (c *Client) Move(uid uint32, folder string) error {
	return c.moveSelected(uid, folder)
}

func (c *Client) MoveFrom(mailbox string, uid uint32, folder string) error {
	return c.MoveMany(mailbox, []uint32{uid}, folder)
}

func (c *Client) CopyTo(mailbox string, uid uint32, folder string) error {
	return c.CopyMany(mailbox, []uint32{uid}, folder)
}

func (c *Client) CopyMany(mailbox string, uids []uint32, folder string) error {
	if err := c.selectMailbox(mailbox); err != nil {
		return err
	}

	for _, chunk := range uidChunks(uids) {
		if err := c.conn.UidCopy(uidSeqSet(chunk), folder); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
		log.Printf("[DEBUG] copied %d messages from %s to %s", len(chunk), mailbox, folder)
	}

	return nil
}

func (c *Client) moveSelected(uid uint32, folder string) error {
	seqSet := &imap.SeqSet{}
	seqSet.AddNum(uid)

	if err := c.conn.UidCopy(seqSet, folder); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	log.Printf("[DEBUG] copied to %s", folder)

	if err := c.storeUIDFlag(seqSet, imap.DeletedFlag); err != nil {
		return fmt.Errorf("store: %w", err)
	}

	if err := c.conn.Expunge(nil); err != nil {
		return fmt.Errorf("expunge: %w", err)
	}

	log.Printf("[DEBUG] moved UID=%d to %s", uid, folder)
	return nil
}

func (c *Client) MarkAsRead(uid uint32) error {
	return c.markAsReadSelected(uid)
}

func (c *Client) MarkAsReadIn(mailbox string, uid uint32) error {
	return c.MarkAsReadMany(mailbox, []uint32{uid})
}

func (c *Client) MarkAsReadMany(mailbox string, uids []uint32) error {
	if err := c.selectMailbox(mailbox); err != nil {
		return err
	}
	for _, chunk := range uidChunks(uids) {
		if err := c.storeUIDFlag(uidSeqSet(chunk), imap.SeenFlag); err != nil {
			return fmt.Errorf("store: %w", err)
		}
	}
	return nil
}

func (c *Client) FlagImportant(mailbox string, uid uint32) error {
	return c.FlagImportantMany(mailbox, []uint32{uid})
}

func (c *Client) FlagImportantMany(mailbox string, uids []uint32) error {
	if err := c.selectMailbox(mailbox); err != nil {
		return err
	}

	for _, chunk := range uidChunks(uids) {
		if err := c.storeUIDFlag(uidSeqSet(chunk), imap.FlaggedFlag); err != nil {
			return fmt.Errorf("store: %w", err)
		}
	}
	return nil
}

func (c *Client) markAsReadSelected(uid uint32) error {
	seqSet := &imap.SeqSet{}
	seqSet.AddNum(uid)

	if err := c.storeUIDFlag(seqSet, imap.SeenFlag); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return nil
}

func (c *Client) Delete(mailbox string, uid uint32) error {
	return c.DeleteMany(mailbox, []uint32{uid})
}

func (c *Client) DeleteMany(mailbox string, uids []uint32) error {
	if err := c.selectMailbox(mailbox); err != nil {
		return err
	}

	for _, chunk := range uidChunks(uids) {
		if err := c.storeUIDFlag(uidSeqSet(chunk), imap.DeletedFlag); err != nil {
			return fmt.Errorf("store: %w", err)
		}
	}

	if err := c.conn.Expunge(nil); err != nil {
		return fmt.Errorf("expunge: %w", err)
	}

	log.Printf("[DEBUG] deleted %d messages from %s", len(uids), mailbox)
	return nil
}

func (c *Client) MoveMany(mailbox string, uids []uint32, folder string) error {
	if err := c.selectMailbox(mailbox); err != nil {
		return err
	}

	for _, chunk := range uidChunks(uids) {
		seqSet := uidSeqSet(chunk)
		if err := c.conn.UidCopy(seqSet, folder); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
		log.Printf("[DEBUG] copied %d messages from %s to %s", len(chunk), mailbox, folder)

		if err := c.storeUIDFlag(seqSet, imap.DeletedFlag); err != nil {
			return fmt.Errorf("store: %w", err)
		}
	}

	if err := c.conn.Expunge(nil); err != nil {
		return fmt.Errorf("expunge: %w", err)
	}

	log.Printf("[DEBUG] moved %d messages from %s to %s", len(uids), mailbox, folder)
	return nil
}

func (c *Client) selectMailbox(mailbox string) error {
	if mailbox == "" {
		mailbox = "INBOX"
	}
	if _, err := c.conn.Select(mailbox, false); err != nil {
		return fmt.Errorf("select %q: %w", mailbox, err)
	}
	return nil
}

func (c *Client) storeUIDFlag(seqSet *imap.SeqSet, flag string) error {
	return c.conn.UidStore(seqSet, imap.AddFlags, []interface{}{flag}, nil)
}

func uidSeqSet(uids []uint32) *imap.SeqSet {
	seqSet := &imap.SeqSet{}
	seqSet.AddNum(uids...)
	return seqSet
}

func uidChunks(uids []uint32) [][]uint32 {
	if len(uids) <= maxUIDBatchSize {
		return [][]uint32{uids}
	}

	chunks := make([][]uint32, 0, (len(uids)+maxUIDBatchSize-1)/maxUIDBatchSize)
	for start := 0; start < len(uids); start += maxUIDBatchSize {
		end := start + maxUIDBatchSize
		if end > len(uids) {
			end = len(uids)
		}
		chunks = append(chunks, uids[start:end])
	}
	return chunks
}

func checkForAttachments(bs *imap.BodyStructure) bool {
	if bs == nil {
		return false
	}

	if bs.MIMEType == "multipart" && bs.MIMESubType == "mixed" {
		for _, child := range bs.Parts {
			if child.Disposition != "" && child.Disposition == "attachment" {
				return true
			}
			if checkForAttachments(child) {
				return true
			}
		}
	}

	if bs.Disposition == "attachment" {
		return true
	}

	return false
}

func hasFlag(flags []string, want string) bool {
	for _, flag := range flags {
		if strings.EqualFold(flag, want) {
			return true
		}
	}
	return false
}
