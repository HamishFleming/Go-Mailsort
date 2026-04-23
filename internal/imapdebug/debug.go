package imapdebug

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type DebugClient struct {
	conn *client.Client
}

func Connect(host string, port int, user, pass string, useTLS bool) (*DebugClient, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[DEBUG] connecting to %s", addr)

	var conn *client.Client
	var err error

	if useTLS {
		conn, err = client.DialTLS(addr, nil)
	} else {
		conn, err = client.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	log.Printf("[DEBUG] connected")

	if err := conn.Login(user, pass); err != nil {
		conn.Logout()
		return nil, fmt.Errorf("login: %w", err)
	}
	log.Printf("[DEBUG] authenticated")

	return &DebugClient{conn: conn}, nil
}

func (d *DebugClient) Close() error {
	return d.conn.Logout()
}

func (d *DebugClient) Capabilities() error {
	caps, err := d.conn.Capability()
	if err != nil {
		return fmt.Errorf("capability: %w", err)
	}
	log.Printf("[INFO] server capabilities:")
	for name, supported := range caps {
		if supported {
			log.Printf("  %s", name)
		}
	}
	return nil
}

func (d *DebugClient) ListMailboxes(pattern string) error {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- d.conn.List("", pattern, mailboxes)
	}()

	log.Printf("[INFO] mailboxes (pattern=%q):", pattern)
	for m := range mailboxes {
		log.Printf("  %s", m.Name)
	}

	return <-done
}

func (d *DebugClient) SelectMailbox(name string) error {
	mbox, err := d.conn.Select(name, false)
	if err != nil {
		return fmt.Errorf("select: %w", err)
	}
	log.Printf("[INFO] mailbox %q: messages=%d, recent=%d, unseen=%d",
		name, mbox.Messages, mbox.Recent, mbox.Unseen)
	return nil
}

func (d *DebugClient) FetchFlags(seqSet string) error {
	seq, err := imap.ParseSeqSet(seqSet)
	if err != nil {
		return fmt.Errorf("parse seqset: %w", err)
	}

	ch := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- d.conn.Fetch(seq, []imap.FetchItem{imap.FetchFlags}, ch)
	}()

	log.Printf("[INFO] flags for messages %s:", seqSet)
	for msg := range ch {
		log.Printf("  UID=%d flags=%v", msg.Uid, msg.Flags)
	}

	return <-done
}

func (d *DebugClient) FetchHeaders(seqSet string) error {
	seq, err := imap.ParseSeqSet(seqSet)
	if err != nil {
		return fmt.Errorf("parse seqset: %w", err)
	}

	ch := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- d.conn.Fetch(seq, []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchUid,
			imap.FetchFlags,
		}, ch)
	}()

	log.Printf("[INFO] headers for messages %s:", seqSet)
	for msg := range ch {
		from := ""
		if len(msg.Envelope.From) > 0 {
			from = msg.Envelope.From[0].Address()
		}
		log.Printf("  UID=%d from=%q subject=%q date=%s flags=%v",
			msg.Uid, from, msg.Envelope.Subject, msg.Envelope.Date, msg.Flags)
	}

	return <-done
}

func (d *DebugClient) StoreFlags(seqSet string, add bool, flags []string) error {
	seq, err := imap.ParseSeqSet(seqSet)
	if err != nil {
		return fmt.Errorf("parse seqset: %w", err)
	}

	var op imap.StoreItem
	if add {
		op = imap.AddFlags
	} else {
		op = imap.RemoveFlags
	}

	if err := d.conn.Store(seq, op, flags, nil); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	log.Printf("[INFO] stored flags %v on %s", flags, seqSet)
	return nil
}

func (d *DebugClient) Search(criteria *imap.SearchCriteria) error {
	ids, err := d.conn.Search(criteria)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	log.Printf("[INFO] matched UIDs: %v", ids)
	return nil
}

func parseSearchCriteria(args []string) (*imap.SearchCriteria, error) {
	criteria := imap.NewSearchCriteria()
	if len(args) == 0 {
		return criteria, nil
	}

	mode := strings.ToUpper(args[0])
	switch mode {
	case "ALL":
		return criteria, nil
	case "UNSEEN":
		criteria.WithoutFlags = append(criteria.WithoutFlags, imap.SeenFlag)
		return criteria, nil
	case "SINCE":
		if len(args) < 2 {
			return nil, fmt.Errorf("usage: mailsort imap-debug search SINCE <date>")
		}
		t, err := time.Parse("2-Jan-2006", args[1])
		if err != nil {
			return nil, fmt.Errorf("invalid date %q (expected 1-Feb-2026): %w", args[1], err)
		}
		criteria.Since = t
		return criteria, nil
	case "KEYWORD":
		if len(args) < 2 {
			return nil, fmt.Errorf("usage: mailsort imap-debug search KEYWORD <flag>")
		}
		criteria.WithFlags = append(criteria.WithFlags, args[1])
		return criteria, nil
	default:
		return nil, fmt.Errorf("unsupported search mode %q (use ALL, UNSEEN, SINCE, KEYWORD)", args[0])
	}
}

func Run(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mailsort imap-debug <subcommand> [args]\nSubcommands: capabilities, list, select, fetch-flags, fetch-headers, search, store")
	}

	host := os.Getenv("IMAP_HOST")
	if host == "" {
		host = "imap.mail.yahoo.com"
	}
	port := 993
	user := os.Getenv("IMAP_USER")
	pass := os.Getenv("IMAP_PASS")
	useTLS := true

	client, err := Connect(host, port, user, pass, useTLS)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close()

	subcmd := args[0]
	switch subcmd {
	case "capabilities":
		return client.Capabilities()
	case "list":
		pattern := "*"
		if len(args) > 1 {
			pattern = args[1]
		}
		return client.ListMailboxes(pattern)
	case "select":
		mbox := "INBOX"
		if len(args) > 1 {
			mbox = args[1]
		}
		return client.SelectMailbox(mbox)
	case "fetch-flags":
		if len(args) < 2 {
			return fmt.Errorf("usage: mailsort imap-debug fetch-flags <seqset>")
		}
		return client.FetchFlags(args[1])
	case "fetch-headers":
		if len(args) < 2 {
			return fmt.Errorf("usage: mailsort imap-debug fetch-headers <seqset>")
		}
		return client.FetchHeaders(args[1])
	case "search":
		criteria, err := parseSearchCriteria(args[1:])
		if err != nil {
			return err
		}
		return client.Search(criteria)
	case "store":
		if len(args) < 4 {
			return fmt.Errorf("usage: mailsort imap-debug store <seqset> <+|-> <flags...>")
		}
		add := args[2] == "+"
		return client.StoreFlags(args[1], add, args[3:])
	default:
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}
