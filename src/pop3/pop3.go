package pop3

import (
	"bytes"

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
		return nil, err
	}

	if err := c.Auth(user, pass); err != nil {
		c.Quit()
		return nil, err
	}

	return &Client{conn: c}, nil
}

func (c *Client) Close() error {
	return c.conn.Quit()
}

func (c *Client) ListMessages() ([]MessageInfo, error) {
	var messages []MessageInfo
	msgs, err := c.conn.Uidl(0)
	if err != nil {
		return nil, err
	}
	for k, v := range msgs {
		// go-pop3 returns 0-based indexes for UIDL; POP3 RETR expects 1-based sequence numbers
		messages = append(messages, MessageInfo{SeqNum: k + 1, UID: v.UID})
	}
	return messages, nil
}

func (c *Client) GetMessage(seqNum int) ([]byte, error) {
	msg, err := c.conn.Retr(seqNum)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := msg.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
