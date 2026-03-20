package main

import (
	"context"
	"errors"
	"jamailify/src/config"
	"jamailify/src/pop3"
	"testing"

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

// Mock Pop3Operations
type mockPop3Client struct {
	messages []pop3.MessageInfo
	rawData  map[int][]byte
}

func (m *mockPop3Client) ListMessages() ([]pop3.MessageInfo, error) {
	return m.messages, nil
}

func (m *mockPop3Client) GetMessage(seqNum int) ([]byte, error) {
	return m.rawData[seqNum], nil
}

func (m *mockPop3Client) Close() error {
	return nil
}

// Mock GmailOperations
type mockGmailClient struct {
	pushedEmails [][]byte
	shouldError  bool
}

func (m *mockGmailClient) PushEmail(ctx context.Context, rawEmail []byte) error {
	if m.shouldError {
		return errors.New("failed to push email")
	}
	m.pushedEmails = append(m.pushedEmails, rawEmail)
	return nil
}

func TestRunSingleSync_HappyPath(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: make(map[string]bool)}
	pop3Client := &mockPop3Client{
		messages: []pop3.MessageInfo{
			{SeqNum: 1, UID: "uid1"},
			{SeqNum: 2, UID: "uid2"},
		},
		rawData: map[int][]byte{
			1: []byte("email1"),
			2: []byte("email2"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, pop3Client, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid1"])
	assert.True(t, db.syncedUIDs["uid2"])
}

func TestRunSingleSync_PartialSync(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: map[string]bool{"uid1": true}}
	pop3Client := &mockPop3Client{
		messages: []pop3.MessageInfo{
			{SeqNum: 1, UID: "uid1"},
			{SeqNum: 2, UID: "uid2"},
			{SeqNum: 3, UID: "uid3"},
		},
		rawData: map[int][]byte{
			2: []byte("email2"),
			3: []byte("email3"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, pop3Client, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid1"])
	assert.True(t, db.syncedUIDs["uid2"])
	assert.True(t, db.syncedUIDs["uid3"])
}

func TestRunSingleSync_ErrorPath(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{}
	db := &mockDB{syncedUIDs: make(map[string]bool)}
	pop3Client := &mockPop3Client{
		messages: []pop3.MessageInfo{
			{SeqNum: 1, UID: "uid1"},
		},
		rawData: map[int][]byte{
			1: []byte("email1"),
		},
	}
	gmailClient := &mockGmailClient{shouldError: true}

	RunSingleSync(context.Background(), account, cfg, db, pop3Client, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 0)
	assert.False(t, db.syncedUIDs["uid1"])
}
