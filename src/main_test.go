package main

import (
	"context"
	"errors"
	"jamailify/src/config"
	"jamailify/src/gmail"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock DBOperations
type mockDB struct {
	syncedUIDs map[string]bool
}

func (m *mockDB) IsSynced(accountName, uid string) (bool, error) {
	return m.syncedUIDs[uid], nil
}

func (m *mockDB) MarkSynced(accountName, uid string) error {
	m.syncedUIDs[uid] = true
	return nil
}

// Mock EmailFetcher
type mockEmailFetcher struct {
	uids          []string
	headers       map[string][]byte
	emails        map[string][]byte
	connectErr    error
	uidsErr       error
	headersErr    error
	emailErr      error
	closeErr      error
	calledClose   bool
	calledConnect bool
}

func (m *mockEmailFetcher) Connect() error {
	m.calledConnect = true
	return m.connectErr
}
func (m *mockEmailFetcher) GetUIDs() ([]string, error) {
	return m.uids, m.uidsErr
}
func (m *mockEmailFetcher) DownloadEmailHeaders(uid string) ([]byte, error) {
	if m.headersErr != nil {
		return nil, m.headersErr
	}
	return m.headers[uid], nil
}
func (m *mockEmailFetcher) DownloadEmail(uid string) ([]byte, error) {
	if m.emailErr != nil {
		return nil, m.emailErr
	}
	return m.emails[uid], nil
}
func (m *mockEmailFetcher) Close() error {
	m.calledClose = true
	return m.closeErr
}

// Mock GmailOperations
type mockGmailClient struct {
	pushedEmails       [][]byte
	existingMessageIDs map[string]bool
	shouldError        bool
}

func (m *mockGmailClient) PushEmail(ctx context.Context, rawEmail []byte) (*gmail.PushResult, error) {
	if m.shouldError {
		return nil, errors.New("failed to push email")
	}
	m.pushedEmails = append(m.pushedEmails, rawEmail)
	return &gmail.PushResult{MessageID: "gmail-message-id", LabelIDs: []string{"INBOX"}}, nil
}

func (m *mockGmailClient) MessageIdExists(ctx context.Context, messageId string) (bool, error) {
	return m.existingMessageIDs[messageId], nil
}

func TestRunSingleSync_HappyPath(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: make(map[string]bool)}
	emailFetcher := &mockEmailFetcher{
		uids: []string{"uid1", "uid2"},
		headers: map[string][]byte{
			"uid1": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
			"uid2": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
		},
		emails: map[string][]byte{
			"uid1": []byte("email1"),
			"uid2": []byte("email2"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, emailFetcher, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid1"])
	assert.True(t, db.syncedUIDs["uid2"])
}

func TestRunSingleSync_PartialSync(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: map[string]bool{"uid1": true}}
	emailFetcher := &mockEmailFetcher{
		uids: []string{"uid1", "uid2", "uid3"},
		headers: map[string][]byte{
			"uid2": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
			"uid3": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
		},
		emails: map[string][]byte{
			"uid2": []byte("email2"),
			"uid3": []byte("email3"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, emailFetcher, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid1"])
	assert.True(t, db.syncedUIDs["uid2"])
	assert.True(t, db.syncedUIDs["uid3"])
}

func TestRunSingleSync_ErrorPath(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: make(map[string]bool)}
	emailFetcher := &mockEmailFetcher{
		uids: []string{"uid1"},
		headers: map[string][]byte{
			"uid1": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
		},
		emails: map[string][]byte{
			"uid1": []byte("email1"),
		},
	}
	gmailClient := &mockGmailClient{shouldError: true}

	RunSingleSync(context.Background(), account, cfg, db, emailFetcher, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 0)
	assert.False(t, db.syncedUIDs["uid1"])
}

func TestRunSingleSync_MaxToCheck(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{MaxMessagesToCheck: 2, SyncWindowDays: 30}
	db := &mockDB{syncedUIDs: make(map[string]bool)}

	emailFetcher := &mockEmailFetcher{
		uids: []string{"uid1", "uid2", "uid3", "uid4", "uid5"},
		headers: map[string][]byte{
			"uid4": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
			"uid5": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"),
		},
		emails: map[string][]byte{
			"uid4": []byte("email4"),
			"uid5": []byte("email5"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, emailFetcher, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.False(t, db.syncedUIDs["uid1"])
	assert.False(t, db.syncedUIDs["uid2"])
	assert.False(t, db.syncedUIDs["uid3"])
	assert.True(t, db.syncedUIDs["uid4"])
	assert.True(t, db.syncedUIDs["uid5"])
}

func TestRunSingleSync_SyncWindow(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{SyncWindowDays: 7}
	db := &mockDB{syncedUIDs: make(map[string]bool)}

	now := time.Now()
	old := now.AddDate(0, 0, -30)

	emailFetcher := &mockEmailFetcher{
		uids: []string{"uid1", "uid2", "uid3"},
		headers: map[string][]byte{
			"uid1": []byte("Date: " + old.Format(time.RFC1123Z) + "\r\n"),
			"uid2": []byte("Date: " + now.Format(time.RFC1123Z) + "\r\n"),
			"uid3": []byte("Date: " + now.Format(time.RFC1123Z) + "\r\n"),
		},
		emails: map[string][]byte{
			"uid2": []byte("email2"),
			"uid3": []byte("email3"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, emailFetcher, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid2"])
	assert.True(t, db.syncedUIDs["uid3"])
	assert.False(t, db.syncedUIDs["uid1"])
}

func TestRunSingleSync_MessageIdDeduplication(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: make(map[string]bool)}
	emailFetcher := &mockEmailFetcher{
		uids: []string{"uid1", "uid2"},
		headers: map[string][]byte{
			"uid1": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\nMessage-ID: <existing-id>\r\n"),
			"uid2": []byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\nMessage-ID: <new-id>\r\n"),
		},
		emails: map[string][]byte{
			"uid1": []byte("email1"),
			"uid2": []byte("email2"),
		},
	}
	gmailClient := &mockGmailClient{
		existingMessageIDs: map[string]bool{"<existing-id>": true},
	}

	RunSingleSync(context.Background(), account, cfg, db, emailFetcher, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 1)
	assert.Equal(t, "email2", string(gmailClient.pushedEmails[0]))
	assert.True(t, db.syncedUIDs["uid1"])
	assert.True(t, db.syncedUIDs["uid2"])
}
