package imap

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"jamailify/src/config"
	"jamailify/src/oauth"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
	"github.com/emersion/go-sasl"
)

// IMAPClient implements the EmailFetcher interface for IMAP servers.
type IMAPClient struct {
	cfg        *config.Account
	client     *client.Client
	normalizer *exchangeResponseNormalizerConn
}

// NewIMAPClient creates a new IMAP client.
func NewIMAPClient(cfg *config.Account) (*IMAPClient, error) {
	return &IMAPClient{cfg: cfg}, nil
}

// Connect connects to the IMAP server and authenticates.
func (c *IMAPClient) Connect() error {
	log.Printf("Connecting to IMAP server: %s", c.cfg.Server)
	conn, err := c.dial()
	if err != nil {
		return err
	}

	c.client, err = client.New(conn)
	if err != nil {
		return fmt.Errorf("failed to initialize IMAP client: %w", err)
	}

	if normalizer, ok := conn.(*exchangeResponseNormalizerConn); ok {
		c.normalizer = normalizer
	}

	if err := c.authenticate(); err != nil {
		return err
	}

	log.Println("Connected to IMAP server")

	// Select INBOX
	if _, err := c.client.Select("INBOX", false); err != nil {
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	return nil
}

func (c *IMAPClient) dial() (net.Conn, error) {
	if c.cfg.NoTls {
		conn, err := net.Dial("tcp", c.cfg.Server)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
		}
		return newExchangeResponseNormalizerConn(conn), nil
	}

	host, _, err := net.SplitHostPort(c.cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid IMAP server address %q: %w", c.cfg.Server, err)
	}

	tlsConn, err := tls.Dial("tcp", c.cfg.Server, &tls.Config{ServerName: host})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	return newExchangeResponseNormalizerConn(tlsConn), nil
}

// authenticate handles IMAP authentication using password or Microsoft OAuth2.
func (c *IMAPClient) authenticate() error {
	if c.cfg.AuthMethod == "oauth2" {
		accessToken, err := oauth.GetMSAccessToken(c.cfg.MSClientID, c.cfg.MSClientSecret, c.cfg.MSRefreshToken)
		if err != nil {
			return fmt.Errorf("IMAP OAuth2 authentication failed — could not obtain access token: %w", err)
		}
		if err := c.authenticateXOAUTH2(accessToken); err != nil {
			return fmt.Errorf("IMAP OAuth2 SASL authentication failed: %w", err)
		}
		c.disableResponseNormalization()
		return nil
	}

	// Default: password-based login
	if err := c.client.Login(c.cfg.User, c.cfg.Pass); err != nil {
		return fmt.Errorf("IMAP password login failed: %w", err)
	}
	c.disableResponseNormalization()
	return nil
}

func (c *IMAPClient) disableResponseNormalization() {
	if c.normalizer != nil {
		c.normalizer.DisableNormalization()
	}
}

func (c *IMAPClient) authenticateXOAUTH2(accessToken string) error {
	supported, err := c.client.SupportAuth("XOAUTH2")
	if err != nil {
		return fmt.Errorf("check XOAUTH2 support: %w", err)
	}
	if !supported {
		return fmt.Errorf("server does not support XOAUTH2")
	}

	saslClient := newXOAUTH2Client(c.cfg.User, accessToken)
	mechanism, initialResponse, err := saslClient.Start()
	if err != nil {
		return fmt.Errorf("start XOAUTH2 exchange: %w", err)
	}

	cmd := &commands.Authenticate{Mechanism: mechanism}
	initialResponseInline, err := c.client.Support("SASL-IR")
	if err != nil {
		return fmt.Errorf("check SASL-IR support: %w", err)
	}
	if initialResponseInline {
		cmd.InitialResponse = initialResponse
	}

	res := &responses.Authenticate{
		Mechanism: saslClient,
		RepliesCh: make(chan []byte, 10),
	}
	if !initialResponseInline {
		res.InitialResponse = initialResponse
	}

	status, err := c.client.Execute(cmd, res)
	if err != nil {
		return err
	}
	if err := status.Err(); err != nil {
		return err
	}
	if err := exchangeAuthStatusError(status); err != nil {
		return err
	}

	c.client.SetState(imap.AuthenticatedState, nil)
	if _, err := c.client.Capability(); err != nil {
		return fmt.Errorf("refresh capabilities after XOAUTH2 auth: %w", err)
	}

	return nil
}

// GetUIDs returns a slice of all unique IDs currently in the INBOX.
func (c *IMAPClient) GetUIDs() ([]string, error) {
	// Search for all messages
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.DeletedFlag}
	uids, err := c.client.UidSearch(criteria)
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
	uid, err := strconv.ParseUint(uidStr, 10, 32)
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

type xoauth2Client struct {
	username string
	token    string
}

func newXOAUTH2Client(username, token string) sasl.Client {
	return &xoauth2Client{
		username: username,
		token:    token,
	}
}

func (c *xoauth2Client) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte("user=" + c.username + "\x01auth=Bearer " + c.token + "\x01\x01")
	return mech, ir, nil
}

func (c *xoauth2Client) Next(challenge []byte) (response []byte, err error) {
	return nil, sasl.ErrUnexpectedServerChallenge
}

func exchangeAuthStatusError(status *imap.StatusResp) error {
	if status == nil {
		return nil
	}

	code := string(status.Code)
	if !strings.HasPrefix(code, "ERROR=") {
		return nil
	}

	parts := []string{code}
	for _, argument := range status.Arguments {
		text, ok := argument.(string)
		if !ok || text == "" {
			continue
		}
		parts = append(parts, text)
	}

	return fmt.Errorf("server reported authentication failure: %s", strings.Join(parts, " "))
}
