package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// a valid config file for testing
	validConfigContent := `{
		"poll_interval_minutes": 15,
		"google_client_id": "google_id",
		"google_client_secret": "google_secret",
		"accounts": [
			{
				"name": "account1",
				"pop3_server": "pop.example.com:995",
				"pop3_user": "user1",
				"pop3_pass": "pass1",
				"gmail_refresh_token": "token1"
			}
		]
	}`
	validConfigFile, err := os.CreateTemp("", "valid_config.json")
	require.NoError(t, err)
	defer os.Remove(validConfigFile.Name())
	_, err = validConfigFile.WriteString(validConfigContent)
	require.NoError(t, err)
	validConfigFile.Close()

	// a malformed config file for testing
	malformedConfigContent := `{
		"poll_interval_minutes": 15,
		"google_client_id": "google_id",
		"google_client_secret": "google_secret",
		"accounts": [
			{
				"name": "account1",
				"pop3_server": "pop.example.com:995",
				"pop3_user": "user1",
				"pop3_pass": "pass1",
				"gmail_refresh_token": "token1"
			}
		]
	`
	malformedConfigFile, err := os.CreateTemp("", "malformed_config.json")
	require.NoError(t, err)
	defer os.Remove(malformedConfigFile.Name())
	_, err = malformedConfigFile.WriteString(malformedConfigContent)
	require.NoError(t, err)
	malformedConfigFile.Close()

	// a config file with missing fields for testing
	missingFieldsConfigContent := `{
		"poll_interval_minutes": 15,
		"google_client_id": "google_id",
		"google_client_secret": "google_secret",
		"accounts": [
			{
				"name": "account1",
				"pop3_user": "user1",
				"pop3_pass": "pass1"
			}
		]
	}`

	missingFieldsConfigFile, err := os.CreateTemp("", "missing_fields_config.json")
	require.NoError(t, err)
	defer os.Remove(missingFieldsConfigFile.Name())
	_, err = missingFieldsConfigFile.WriteString(missingFieldsConfigContent)
	require.NoError(t, err)
	missingFieldsConfigFile.Close()

	testCases := []struct {
		name          string
		path          string
		expectError   bool
		expectedError string
		check         func(t *testing.T, config *Config)
	}{
		{
			name:        "Valid configuration",
			path:        validConfigFile.Name(),
			expectError: false,
			check: func(t *testing.T, config *Config) {
				assert.Equal(t, 15, config.PollIntervalMinutes)
				assert.Equal(t, "google_id", config.GoogleClientID)
				assert.Equal(t, "google_secret", config.GoogleClientSecret)
				require.Len(t, config.Accounts, 1)
				assert.Equal(t, "account1", config.Accounts[0].Name)
				assert.Equal(t, "pop.example.com:995", config.Accounts[0].Pop3Server)
				assert.Equal(t, "user1", config.Accounts[0].Pop3User)
				assert.Equal(t, "pass1", config.Accounts[0].Pop3Pass)
				assert.Equal(t, "token1", config.Accounts[0].GmailRefreshToken)
			},
		},
		{
			name:          "Missing file",
			path:        "non_existent_file.json",
			expectError:   true,
			expectedError: "",
		},
		{
			name:        "Malformed JSON",
			path:        malformedConfigFile.Name(),
			expectError: true,
		},
		{
			name:        "Missing required fields",
			path:        missingFieldsConfigFile.Name(),
			expectError: true, 
			expectedError: "pop3_server is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadConfig(tc.path)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(t, err)
				if tc.check != nil {
					tc.check(t, config)
				}
			}
		})
	}
}
