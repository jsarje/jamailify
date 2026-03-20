package gmail

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

// MockInsertCall is a mock for the InsertCall interface
type MockInsertCall struct {
	mock.Mock
}

func (m *MockInsertCall) Do(opts ...googleapi.CallOption) (*gmail.Message, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Message), args.Error(1)
}

// MockMessageInserter is a mock for the MessageInserter interface
type MockMessageInserter struct {
	mock.Mock
}

func (m *MockMessageInserter) Insert(userId string, message *gmail.Message) InsertCall {
	args := m.Called(userId, message)
	return args.Get(0).(InsertCall)
}

func TestPushEmail(t *testing.T) {
	mockInserter := new(MockMessageInserter)
	mockInsertCall := new(MockInsertCall)

	client := &Client{inserter: mockInserter}

	rawEmail := []byte("From: from@example.com\nTo: to@example.com\nSubject: Test\n\nTest Body")
	expectedEncodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)

	// Set up the expectation.
	mockInserter.On("Insert", "me", mock.MatchedBy(func(msg *gmail.Message) bool {
		return msg.Raw == expectedEncodedEmail
	})).Return(mockInsertCall)
	mockInsertCall.On("Do").Return(&gmail.Message{}, nil)

	// Call the method.
	err := client.PushEmail(context.Background(), rawEmail)

	// Assert that the expectations were met.
	assert.NoError(t, err)
	mockInserter.AssertExpectations(t)
	mockInsertCall.AssertExpectations(t)
}
