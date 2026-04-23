package imapclient

import (
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Email struct {
	Uid            uint32
	From           string
	Subject        string
	Body           string
	Date           time.Time
	Size           uint32
	HasAttachments  bool
}

type Config struct {
	Host   string
	Port   int
	User   string
	Pass   string
	UseTLS bool
	Mailbox string
}

type Provider interface {
	Connect(cfg *Config) (*Client, error)
}

type Client struct {
	cfg  *Config
	conn *client.Client
}

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
			From:           from,
			Subject:        msg.Envelope.Subject,
			Date:           msg.Envelope.Date,
			Size:           msg.Size,
			HasAttachments: hasAttachments,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	log.Printf("[DEBUG] fetched %d emails", len(emails))
	return emails, nil
}

func (c *Client) Move(uid uint32, folder string) error {
	seqSet := &imap.SeqSet{}
	seqSet.AddNum(uid)

	if err := c.conn.UidCopy(seqSet, folder); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	log.Printf("[DEBUG] copied to %s", folder)

	if err := c.conn.UidStore(seqSet, imap.AddFlags, imap.DeletedFlag, nil); err != nil {
		return fmt.Errorf("store: %w", err)
	}

	if err := c.conn.Expunge(nil); err != nil {
		return fmt.Errorf("expunge: %w", err)
	}

	log.Printf("[DEBUG] moved UID=%d to %s", uid, folder)
	return nil
}

func (c *Client) MarkAsRead(uid uint32) error {
	seqSet := &imap.SeqSet{}
	seqSet.AddNum(uid)

	if err := c.conn.UidStore(seqSet, imap.AddFlags, imap.SeenFlag, nil); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return nil
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