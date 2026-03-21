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

// MockImportCall is a mock for the ImportCall interface
type MockImportCall struct {
	mock.Mock
}

func (m *MockImportCall) Do(opts ...googleapi.CallOption) (*gmail.Message, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Message), args.Error(1)
}

// MockMessageImporter is a mock for the MessageImporter interface
type MockMessageImporter struct {
	mock.Mock
}

func (m *MockMessageImporter) Import(userId string, message *gmail.Message) ImportCall {
	args := m.Called(userId, message)
	return args.Get(0).(ImportCall)
}

func TestPushEmail(t *testing.T) {
	mockImporter := new(MockMessageImporter)
	mockImportCall := new(MockImportCall)

	client := &Client{importer: mockImporter}

	rawEmail := []byte("From: from@example.com\nTo: to@example.com\nSubject: Test\n\nTest Body")
	expectedEncodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)

	// Set up the expectation.
	mockImporter.On("Import", "me", mock.MatchedBy(func(msg *gmail.Message) bool {
		return msg.Raw == expectedEncodedEmail
	})).Return(mockImportCall)
	mockImportCall.On("Do").Return(&gmail.Message{}, nil)

	// Call the method.
	err := client.PushEmail(context.Background(), rawEmail)

	// Assert that the expectations were met.
	assert.NoError(t, err)
	mockImporter.AssertExpectations(t)
	mockImportCall.AssertExpectations(t)
}
