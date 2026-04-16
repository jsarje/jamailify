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

func (m *MockImportCall) Context(ctx context.Context) ImportCall {
	args := m.Called(ctx)
	return args.Get(0).(ImportCall)
}

func (m *MockImportCall) Fields(s ...googleapi.Field) ImportCall {
	args := m.Called()
	return args.Get(0).(ImportCall)
}

func (m *MockImportCall) InternalDateSource(source string) ImportCall {
	args := m.Called(source)
	return args.Get(0).(ImportCall)
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

type MockGetCall struct {
	mock.Mock
}

func (m *MockGetCall) Context(ctx context.Context) GetCall {
	args := m.Called(ctx)
	return args.Get(0).(GetCall)
}

func (m *MockGetCall) Fields(s ...googleapi.Field) GetCall {
	args := m.Called()
	return args.Get(0).(GetCall)
}

func (m *MockGetCall) Format(format string) GetCall {
	args := m.Called(format)
	return args.Get(0).(GetCall)
}

func (m *MockGetCall) Do(opts ...googleapi.CallOption) (*gmail.Message, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Message), args.Error(1)
}

type MockMessageGetter struct {
	mock.Mock
}

func (m *MockMessageGetter) Get(userId, id string) GetCall {
	args := m.Called(userId, id)
	return args.Get(0).(GetCall)
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

func TestPushEmail_DefaultDoesNotFetchMetadataAfterImport(t *testing.T) {
	mockImporter := new(MockMessageImporter)
	mockImportCall := new(MockImportCall)

	client := &Client{importer: mockImporter}

	rawEmail := []byte("Message-ID: <abc@example.com>\nFrom: from@example.com\nTo: to@example.com\nSubject: Test\n\nTest Body")
	expectedEncodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)

	// Set up the importer expectation.
	mockImporter.On("Import", "me", mock.MatchedBy(func(msg *gmail.Message) bool {
		return msg.Raw == expectedEncodedEmail
	})).Return(mockImportCall)
	mockImportCall.On("Context", mock.Anything).Return(mockImportCall)
	mockImportCall.On("Fields").Return(mockImportCall)
	mockImportCall.On("InternalDateSource", "receivedTime").Return(mockImportCall)
	mockImportCall.On("Do").Return(&gmail.Message{Id: "gmail-message-id", LabelIds: []string{"INBOX"}, InternalDate: 1234}, nil)

	// Call the method.
	result, err := client.PushEmail(context.Background(), rawEmail)

	// Assert that the expectations were met.
	assert.NoError(t, err)
	assert.Equal(t, "gmail-message-id", result.MessageID)
	assert.Equal(t, []string{"INBOX"}, result.LabelIDs)
	assert.EqualValues(t, 1234, result.InternalDate)
	mockImporter.AssertExpectations(t)
	mockImportCall.AssertExpectations(t)
}

func TestPushEmail_FetchesMetadataAfterImportWhenEnabled(t *testing.T) {
	mockImporter := new(MockMessageImporter)
	mockImportCall := new(MockImportCall)
	mockGetter := new(MockMessageGetter)
	mockGetCall := new(MockGetCall)

	client := &Client{importer: mockImporter, getter: mockGetter, fetchMetadataAfterImport: true}

	rawEmail := []byte("Message-ID: <abc@example.com>\nFrom: from@example.com\nTo: to@example.com\nSubject: Test\n\nTest Body")
	expectedEncodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)

	mockImporter.On("Import", "me", mock.MatchedBy(func(msg *gmail.Message) bool {
		return msg.Raw == expectedEncodedEmail
	})).Return(mockImportCall)
	mockImportCall.On("Context", mock.Anything).Return(mockImportCall)
	mockImportCall.On("Fields").Return(mockImportCall)
	mockImportCall.On("InternalDateSource", "receivedTime").Return(mockImportCall)
	mockImportCall.On("Do").Return(&gmail.Message{Id: "gmail-message-id"}, nil)
	mockGetter.On("Get", "me", "gmail-message-id").Return(mockGetCall)
	mockGetCall.On("Context", mock.Anything).Return(mockGetCall)
	mockGetCall.On("Format", "metadata").Return(mockGetCall)
	mockGetCall.On("Fields").Return(mockGetCall)
	mockGetCall.On("Do").Return(&gmail.Message{Id: "gmail-message-id", LabelIds: []string{"INBOX"}, InternalDate: 1234}, nil)

	result, err := client.PushEmail(context.Background(), rawEmail)

	assert.NoError(t, err)
	assert.Equal(t, "gmail-message-id", result.MessageID)
	assert.Equal(t, []string{"INBOX"}, result.LabelIDs)
	assert.EqualValues(t, 1234, result.InternalDate)
	mockImporter.AssertExpectations(t)
	mockImportCall.AssertExpectations(t)
	mockGetter.AssertExpectations(t)
	mockGetCall.AssertExpectations(t)
}
