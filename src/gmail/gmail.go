package gmail

import (
	"context"
	"encoding/base64"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type InsertCall interface {
	Do(opts ...googleapi.CallOption) (*gmail.Message, error)
}

type MessageInserter interface {
	Insert(userId string, message *gmail.Message) InsertCall
}

type gmailAPIAdapter struct {
	service *gmail.Service
}

func (g *gmailAPIAdapter) Insert(userId string, message *gmail.Message) InsertCall {
	return &insertCallAdapter{call: g.service.Users.Messages.Insert(userId, message)}
}

type insertCallAdapter struct {
	call *gmail.UsersMessagesInsertCall
}

func (i *insertCallAdapter) Do(opts ...googleapi.CallOption) (*gmail.Message, error) {
	return i.call.Do(opts...)
}

type Client struct {
	inserter MessageInserter
}

// NewClient creates a Gmail client using the provided OAuth2 credentials.
// The ctx provided will be used for creating the underlying Gmail service.
func NewClient(ctx context.Context, clientID, clientSecret, refreshToken string) (*Client, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailInsertScope},
	}

	token := &oauth2.Token{RefreshToken: refreshToken}

	tokenSource := cfg.TokenSource(ctx, token)

	srv, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}

	return &Client{inserter: &gmailAPIAdapter{service: srv}}, nil
}

// PushEmail pushes a raw RFC2822 email to the authenticated user's mailbox.
// The provided ctx is not currently passed into the underlying API call
// because the gmail insert call builder does not accept a context directly,
// but ctx is used when creating the client (for token source). This
// signature is provided to make callers context-aware and to allow future
// context-aware changes.
func (c *Client) PushEmail(ctx context.Context, rawRFC2822 []byte) error {
	_ = ctx // currently unused but kept for API symmetry
	message := &gmail.Message{Raw: base64.RawURLEncoding.EncodeToString(rawRFC2822)}
	_, err := c.inserter.Insert("me", message).Do()
	if err != nil {
		return fmt.Errorf("push email: %w", err)
	}
	return nil
}
