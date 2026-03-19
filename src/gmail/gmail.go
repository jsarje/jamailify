package gmail

import (
	"context"

	"encoding/base64"

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

func NewClient(clientID, clientSecret, refreshToken string) (*Client, error) {

	config := &oauth2.Config{

		ClientID: clientID,

		ClientSecret: clientSecret,

		Endpoint: google.Endpoint,

		Scopes: []string{gmail.GmailInsertScope},
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

	return &Client{inserter: &gmailAPIAdapter{service: srv}}, nil

}

func (c *Client) PushEmail(rawRFC2822 []byte) error {

	message := &gmail.Message{

		Raw: base64.RawURLEncoding.EncodeToString(rawRFC2822),
	}

	_, err := c.inserter.Insert("me", message).Do()

	return err

}
