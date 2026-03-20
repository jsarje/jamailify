package main

import (
	"context"
	"errors"
	"jamailify/src/config"
	"jamailify/src/pop3"
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

// Mock Pop3Operations
type mockPop3Client struct {
	messages   []pop3.MessageInfo
	rawData    map[int][]byte
	topErrors  map[int]error
	topHeaders map[int]string
}

func (m *mockPop3Client) Stat() (int, error) {
	return len(m.messages), nil
}

func (m *mockPop3Client) UIDLForSeq(seqNum int) (string, error) {
	for _, mi := range m.messages {
		if mi.SeqNum == seqNum {
			return mi.UID, nil
		}
	}
	return "", errors.New("not found")
}

func (m *mockPop3Client) TopMessage(seqNum int) ([]byte, error) {
	if m.topErrors != nil {
		if err, ok := m.topErrors[seqNum]; ok {
			return nil, err
		}
	}
	if m.topHeaders != nil {
		if h, ok := m.topHeaders[seqNum]; ok {
			return []byte(h), nil
		}
	}
	// default: return minimal headers with Date: now
	now := time.Now().Format(time.RFC1123Z)
	hdr := "Date: " + now + "\r\nSubject: test\r\n\r\n"
	return []byte(hdr), nil
}

func (m *mockPop3Client) GetMessage(seqNum int) ([]byte, error) {
	if b, ok := m.rawData[seqNum]; ok {
		return b, nil
	}
	return nil, errors.New("not found")
}

func (m *mockPop3Client) Close() error { return nil }

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

func TestRunSingleSync_MaxToCheck(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{MaxMessagesToCheck: 2, SyncWindowDays: 30}
	db := &mockDB{syncedUIDs: make(map[string]bool)}

	// 5 messages, only last 2 should be inspected
	pop3Client := &mockPop3Client{
		messages: []pop3.MessageInfo{
			{SeqNum: 1, UID: "uid1"},
			{SeqNum: 2, UID: "uid2"},
			{SeqNum: 3, UID: "uid3"},
			{SeqNum: 4, UID: "uid4"},
			{SeqNum: 5, UID: "uid5"},
		},
		rawData: map[int][]byte{
			4: []byte("email4"),
			5: []byte("email5"),
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, pop3Client, gmailClient)

	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid4"]) // seq 4
	assert.True(t, db.syncedUIDs["uid5"]) // seq 5
}

func TestRunSingleSync_TopFallbackAndWindow(t *testing.T) {
	account := config.Account{Name: "test-account"}
	cfg := &config.Config{SyncWindowDays: 7}
	db := &mockDB{syncedUIDs: make(map[string]bool)}

	// seq 3 newest, seq2 will force TOP error and use RETR fallback, seq1 is old and should stop the scan
	now := time.Now()
	old := now.AddDate(0, 0, -30)

	pop3Client := &mockPop3Client{
		messages: []pop3.MessageInfo{
			{SeqNum: 1, UID: "uid1"},
			{SeqNum: 2, UID: "uid2"},
			{SeqNum: 3, UID: "uid3"},
		},
		rawData: map[int][]byte{
			2: []byte("Date: " + now.Format(time.RFC1123Z) + "\r\nSubject: email2\r\n\r\nBody2"),
			3: []byte("email3"),
		},
		topErrors: map[int]error{2: errors.New("TOP not supported")},
		topHeaders: map[int]string{
			1: "Date: " + old.Format(time.RFC1123Z) + "\r\nSubject: old\r\n\r\n",
			3: "Date: " + now.Format(time.RFC1123Z) + "\r\nSubject: new\r\n\r\n",
		},
	}
	gmailClient := &mockGmailClient{}

	RunSingleSync(context.Background(), account, cfg, db, pop3Client, gmailClient)

	// seq3 and seq2 should be pushed; seq1 is old and stops the scan
	assert.Len(t, gmailClient.pushedEmails, 2)
	assert.True(t, db.syncedUIDs["uid3"])
	assert.True(t, db.syncedUIDs["uid2"])
	assert.False(t, db.syncedUIDs["uid1"])
}
