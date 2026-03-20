package pop3

import (
	"bytes"
	"net"
	"strconv"

	"github.com/knadh/go-pop3"
)

type Client struct {
	conn *pop3.Conn
}

type MessageInfo struct {
	SeqNum int
	UID    string
}

func NewClient(server, user, pass string) (*Client, error) {
	host, portStr, err := net.SplitHostPort(server)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	p := pop3.New(pop3.Opt{
		Host:       host,
		Port:       port,
		TLSEnabled: true,
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
		messages = append(messages, MessageInfo{SeqNum: k, UID: v.UID})
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
