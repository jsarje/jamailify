package pop3

import (
	"bytes"
	"fmt"

	"github.com/knadh/go-pop3"
)

type Client struct {
	conn *pop3.Conn
}

type MessageInfo struct {
	SeqNum int
	UID    string
}

func NewClient(host string, port int, user, pass string, useTLS bool) (*Client, error) {
	p := pop3.New(pop3.Opt{
		Host:       host,
		Port:       port,
		TLSEnabled: useTLS,
	})

	c, err := p.NewConn()
	if err != nil {
		return nil, fmt.Errorf("connect to %s:%d: %w", host, port, err)
	}

	if c == nil {
		return nil, fmt.Errorf("pop3: connection returned nil")
	}

	if err := c.Auth(user, pass); err != nil {
		c.Quit()
		return nil, fmt.Errorf("auth for user %s: %w", user, err)
	}

	return &Client{conn: c}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Quit()
}

func (c *Client) ListMessages() ([]MessageInfo, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("pop3: client is not connected")
	}

	var messages []MessageInfo
	msgs, err := c.conn.Uidl(0)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	if len(msgs) == 0 {
		return messages, nil
	}
	for k, v := range msgs {
		// go-pop3 returns 0-based indexes for UIDL; POP3 RETR expects 1-based sequence numbers
		messages = append(messages, MessageInfo{SeqNum: k + 1, UID: v.UID})
	}
	return messages, nil
}

func (c *Client) GetMessage(seqNum int) ([]byte, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("pop3: client is not connected")
	}

	msg, err := c.conn.Retr(seqNum)
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
