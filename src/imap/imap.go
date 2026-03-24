package imap

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"jamailify/src/config"
	"jamailify/src/fetcher"
)

// IMAPClient implements the EmailFetcher interface for IMAP servers.
type IMAPClient struct {
	cfg    *config.Account
	client *client.Client
}

// NewIMAPClient creates a new IMAP client.
func NewIMAPClient(cfg *config.Account) (fetcher.EmailFetcher, error) {
	return &IMAPClient{cfg: cfg}, nil
}

// Connect connects to the IMAP server and authenticates.
func (c *IMAPClient) Connect() error {
	log.Printf("Connecting to IMAP server: %s", c.cfg.Server)
	// Connect to the server
	cl, err := client.DialTLS(c.cfg.Server, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	c.client = cl

	// Login
	if err := c.client.Login(c.cfg.User, c.cfg.Pass); err != nil {
		return fmt.Errorf("failed to login to IMAP server: %w", err)
	}
	log.Println("Connected to IMAP server")

	// Select INBOX
	if _, err := c.client.Select("INBOX", false); err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	return nil
}

// GetUIDs returns a slice of all unique IDs currently in the INBOX.
func (c *IMAPClient) GetUIDs() ([]string, error) {
	// Search for all messages
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.DeletedFlag}
	uids, err := c.client.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search for messages: %w", err)
	}

	var uidStrs []string
	for _, uid := range uids {
		uidStrs = append(uidStrs, fmt.Sprintf("%d", uid))
	}

	return uidStrs, nil
}

// DownloadEmailHeaders downloads only the headers for a specific UID.
func (c *IMAPClient) DownloadEmailHeaders(uid string) ([]byte, error) {
	return c.download(uid, true)
}

// DownloadEmail downloads the raw RFC 2822 bytes for a specific UID.
func (c *IMAPClient) DownloadEmail(uid string) ([]byte, error) {
	return c.download(uid, false)
}

func (c *IMAPClient) download(uidStr string, headersOnly bool) ([]byte, error) {
	seqset := new(imap.SeqSet)
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UID: %s", uidStr)
	}
	seqset.AddNum(uint32(uid))

	section := imap.BodySectionName{Peek: true}
	if headersOnly {
		section.Specifier = imap.HeaderSpecifier
	}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.client.UidFetch(seqset, []imap.FetchItem{section.FetchItem()}, messages)
	}()

	msg := <-messages
	if msg == nil {
		return nil, fmt.Errorf("failed to fetch message with UID: %s", uidStr)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch message with UID %s: %w", uidStr, err)
	}

	r := msg.GetBody(&section)
	if r == nil {
		return nil, fmt.Errorf("failed to get message body for UID: %s", uidStr)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, fmt.Errorf("failed to read message body for UID %s: %w", uidStr, err)
	}

	return buf.Bytes(), nil
}

// Close closes the connection cleanly.
func (c *IMAPClient) Close() error {
	if c.client != nil {
		return c.client.Logout()
	}
	return nil
}
