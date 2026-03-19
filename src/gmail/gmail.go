package gmail

import (
	"context"
	"encoding/base64"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Client struct {
	service *gmail.Service
}

func NewClient(clientID, clientSecret, refreshToken string) (*Client, error) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailInsertScope},
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	ctx := context.Background()
	tokenSource := config.TokenSource(ctx, token)

	srv, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	return &Client{service: srv}, nil
}

func (c *Client) PushEmail(rawRFC2822 []byte) error {
	message := &gmail.Message{
		Raw: base64.RawURLEncoding.EncodeToString(rawRFC2822),
	}

	_, err := c.service.Users.Messages.Insert("me", message).Do()
	return err
}