package pop3

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/knadh/go-pop3"
	"jamailify/src/config"
	"jamailify/src/fetcher"
)

// POP3Client implements the EmailFetcher interface for POP3 servers.
type POP3Client struct {
	cfg    *config.Account
	client *pop3.Conn
}

// NewPOP3Client creates a new POP3 client.
func NewPOP3Client(cfg *config.Account) (fetcher.EmailFetcher, error) {
	return &POP3Client{cfg: cfg}, nil
}

func (c *POP3Client) Connect() error {
	host, portStr, _ := strings.Cut(c.cfg.Server, ":")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port in server address: %s", c.cfg.Server)
	}

	p := pop3.New(pop3.Opt{
		Host:       host,
		Port:       port,
		TLSEnabled: true,
	})

	conn, err := p.NewConn()
	if err != nil {
		return fmt.Errorf("connect to %s:%d: %w", host, port, err)
	}

	if conn == nil {
		return fmt.Errorf("pop3: connection returned nil")
	}

	if err := conn.Auth(c.cfg.User, c.cfg.Pass); err != nil {
		if qerr := conn.Quit(); qerr != nil {
			return fmt.Errorf("auth for user %s: %v; quit error: %w", c.cfg.User, err, qerr)
		}
		return fmt.Errorf("auth for user %s: %w", c.cfg.User, err)
	}

	c.client = conn
	return nil
}

func (c *POP3Client) Close() error {
	if c.client != nil {
		return c.client.Quit()
	}
	return nil
}

func (c *POP3Client) GetUIDs() ([]string, error) {
	msgs, err := c.client.Uidl(0)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	var uids []string
	for _, m := range msgs {
		uids = append(uids, m.UID)
	}
	return uids, nil
}

func (c *POP3Client) DownloadEmailHeaders(uid string) ([]byte, error) {
	seqNum, err := c.getSeqNumFromUID(uid)
	if err != nil {
		return nil, err
	}
	msg, err := c.client.Top(seqNum, 0)
	if err != nil {
		return nil, fmt.Errorf("top seq %d: %w", seqNum, err)
	}
	if msg == nil {
		return nil, fmt.Errorf("pop3: top returned nil for seq %d", seqNum)
	}
	var buf bytes.Buffer
	if err := msg.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *POP3Client) DownloadEmail(uid string) ([]byte, error) {
	seqNum, err := c.getSeqNumFromUID(uid)
	if err != nil {
		return nil, err
	}
	msg, err := c.client.Retr(seqNum)
	if err != nil {
		return nil, fmt.Errorf("retr seq %d: %w", seqNum, err)
	}
	if msg == nil {
		return nil, fmt.Errorf("pop3: received nil message for seq %d", seqNum)
	}

	var buf bytes.Buffer
	if err := msg.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *POP3Client) getSeqNumFromUID(uid string) (int, error) {
	msgs, err := c.client.Uidl(0)
	if err != nil {
		return 0, fmt.Errorf("uidl: %w", err)
	}
	for i, m := range msgs {
		if m.UID == uid {
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("uid %s not found", uid)
}
