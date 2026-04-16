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

// Gmail messages.import is authorized with gmail.insert or gmail.modify.
// There is no separate gmail.import OAuth scope.

type ImportCall interface {
	Context(ctx context.Context) ImportCall
	Fields(s ...googleapi.Field) ImportCall
	InternalDateSource(source string) ImportCall
	Do(opts ...googleapi.CallOption) (*gmail.Message, error)
}

type MessageImporter interface {
	Import(userId string, message *gmail.Message) ImportCall
}

type GetCall interface {
	Context(ctx context.Context) GetCall
	Fields(s ...googleapi.Field) GetCall
	Format(format string) GetCall
	Do(opts ...googleapi.CallOption) (*gmail.Message, error)
}

type MessageGetter interface {
	Get(userId, id string) GetCall
}

type gmailAPIAdapter struct {
	service *gmail.Service
}

func (g *gmailAPIAdapter) Import(userId string, message *gmail.Message) ImportCall {
	return &importCallAdapter{call: g.service.Users.Messages.Import(userId, message)}
}

func (g *gmailAPIAdapter) List(userId string) ListCall {
	return &listCallAdapter{call: g.service.Users.Messages.List(userId)}
}

func (g *gmailAPIAdapter) Get(userId, id string) GetCall {
	return &getCallAdapter{call: g.service.Users.Messages.Get(userId, id)}
}

type importCallAdapter struct {
	call *gmail.UsersMessagesImportCall
}

type getCallAdapter struct {
	call *gmail.UsersMessagesGetCall
}

func (i *importCallAdapter) Context(ctx context.Context) ImportCall {
	return &importCallAdapter{call: i.call.Context(ctx)}
}

func (i *importCallAdapter) Fields(s ...googleapi.Field) ImportCall {
	return &importCallAdapter{call: i.call.Fields(s...)}
}

func (i *importCallAdapter) InternalDateSource(source string) ImportCall {
	return &importCallAdapter{call: i.call.InternalDateSource(source)}
}

func (i *importCallAdapter) Do(opts ...googleapi.CallOption) (*gmail.Message, error) {
	return i.call.Do(opts...)
}

func (g *getCallAdapter) Context(ctx context.Context) GetCall {
	return &getCallAdapter{call: g.call.Context(ctx)}
}

func (g *getCallAdapter) Fields(s ...googleapi.Field) GetCall {
	return &getCallAdapter{call: g.call.Fields(s...)}
}

func (g *getCallAdapter) Format(format string) GetCall {
	return &getCallAdapter{call: g.call.Format(format)}
}

func (g *getCallAdapter) Do(opts ...googleapi.CallOption) (*gmail.Message, error) {
	return g.call.Do(opts...)
}

// ListCall abstracts Users.Messages.List so we can set query params and Do.
type ListCall interface {
	Q(q string) ListCall
	MaxResults(max int64) ListCall
	Do(opts ...googleapi.CallOption) (*gmail.ListMessagesResponse, error)
}

type MessageLister interface {
	List(userId string) ListCall
}

type listCallAdapter struct {
	call *gmail.UsersMessagesListCall
}

func (l *listCallAdapter) Q(q string) ListCall {
	return &listCallAdapter{call: l.call.Q(q)}
}

func (l *listCallAdapter) MaxResults(max int64) ListCall {
	return &listCallAdapter{call: l.call.MaxResults(max)}
}

func (l *listCallAdapter) Do(opts ...googleapi.CallOption) (*gmail.ListMessagesResponse, error) {
	return l.call.Do(opts...)
}

type Client struct {
	importer MessageImporter
	getter   MessageGetter
	lister   MessageLister
}

type PushResult struct {
	MessageID    string
	ThreadID     string
	LabelIDs     []string
	InternalDate int64
}

// NewClient creates a Gmail client using the provided OAuth2 credentials.
// The ctx provided will be used for creating the underlying Gmail service.
func NewClient(ctx context.Context, clientID, clientSecret, refreshToken string) (*Client, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailInsertScope, gmail.GmailReadonlyScope},
	}

	token := &oauth2.Token{RefreshToken: refreshToken}

	tokenSource := cfg.TokenSource(ctx, token)

	srv, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}

	adapter := &gmailAPIAdapter{service: srv}
	return &Client{importer: adapter, getter: adapter, lister: adapter}, nil
}

// PushEmail pushes a raw RFC2822 email to the authenticated user's mailbox.
// The import uses Gmail's received time so newly imported messages sort near
// the top instead of being ordered by the original Date header.
func (c *Client) PushEmail(ctx context.Context, rawRFC2822 []byte) (*PushResult, error) {
	message := &gmail.Message{Raw: base64.RawURLEncoding.EncodeToString(rawRFC2822)}
	imported, err := c.importer.Import("me", message).
		Context(ctx).
		Fields(googleapi.Field("id"), googleapi.Field("threadId"), googleapi.Field("labelIds"), googleapi.Field("internalDate")).
		InternalDateSource("receivedTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("push email: %w", err)
	}
	if imported.Id == "" {
		return nil, fmt.Errorf("push email: gmail import returned empty message id")
	}

	resolved := imported
	if c.getter != nil {
		resolved, err = c.getter.Get("me", imported.Id).
			Context(ctx).
			Format("metadata").
			Fields(googleapi.Field("id"), googleapi.Field("threadId"), googleapi.Field("labelIds"), googleapi.Field("internalDate")).
			Do()
		if err != nil {
			return nil, fmt.Errorf("get imported message metadata: %w", err)
		}
	}
	return &PushResult{
		MessageID:    resolved.Id,
		ThreadID:     resolved.ThreadId,
		LabelIDs:     append([]string(nil), resolved.LabelIds...),
		InternalDate: resolved.InternalDate,
	}, nil
}

func (c *Client) MessageIdExists(ctx context.Context, messageId string) (bool, error) {
	_ = ctx // currently unused but kept for API symmetry
	if c.lister == nil {
		return false, nil
	}
	q := fmt.Sprintf("rfc822msgid:%s", messageId)
	res, err := c.lister.List("me").Q(q).MaxResults(1).Do()
	if err != nil {
		return false, fmt.Errorf("check messageIdExists: %w", err)
	}
	return len(res.Messages) > 0, nil
}
