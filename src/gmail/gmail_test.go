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

// MockListCall is a mock for the ListCall interface
type MockListCall struct {
	mock.Mock
}

func (m *MockListCall) Q(q string) ListCall {
	m.Called(q)
	return m
}

func (m *MockListCall) MaxResults(max int64) ListCall {
	m.Called(max)
	return m
}

func (m *MockListCall) Do(opts ...googleapi.CallOption) (*gmail.ListMessagesResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.ListMessagesResponse), args.Error(1)
}

// MockMessageLister is a mock for the MessageLister interface
type MockMessageLister struct {
	mock.Mock
}

func (m *MockMessageLister) List(userId string) ListCall {
	args := m.Called(userId)
	return args.Get(0).(ListCall)
}

func TestPushEmail(t *testing.T) {
	mockImporter := new(MockMessageImporter)
	mockImportCall := new(MockImportCall)
	// Also mock the lister so PushEmail's pre-import dedupe path is exercised.
	mockLister := new(MockMessageLister)
	mockListCall := new(MockListCall)

	client := &Client{importer: mockImporter, lister: mockLister}

	rawEmail := []byte("Message-ID: <abc@example.com>\nFrom: from@example.com\nTo: to@example.com\nSubject: Test\n\nTest Body")
	expectedEncodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)

	// Set up the lister expectations: search returns no existing messages.
	mockLister.On("List", "me").Return(mockListCall)
	mockListCall.On("Q", mock.Anything).Return(mockListCall)
	mockListCall.On("MaxResults", int64(1)).Return(mockListCall)
	mockListCall.On("Do").Return(&gmail.ListMessagesResponse{Messages: nil}, nil)

	// Set up the importer expectation: will be called after search returns no match.
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
