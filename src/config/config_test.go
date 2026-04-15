package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name          string
		path          func(t *testing.T) string
		expectError   bool
		expectedError string
		check         func(t *testing.T, config *Config)
	}{
		{
			name: "Valid configuration",
			path: func(t *testing.T) string {
				validConfigContent := `{
					"poll_interval_minutes": 15,
					"google_client_id": "google_id",
					"google_client_secret": "google_secret",
					"accounts": [
						{
							"name": "account1",
							"protocol": "pop3",
							"server": "pop.example.com:995",
							"user": "user1",
							"pass": "pass1",
							"gmail_refresh_token": "token1"
						}
					]
				}`
				validConfigFile, err := os.CreateTemp("", "valid_config.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(validConfigFile.Name()) })
				_, err = validConfigFile.WriteString(validConfigContent)
				require.NoError(t, err)
				validConfigFile.Close()
				return validConfigFile.Name()
			},
			expectError: false,
			check: func(t *testing.T, config *Config) {
				assert.Equal(t, 15, config.PollIntervalMinutes)
				assert.Equal(t, "google_id", config.GoogleClientID)
				assert.Equal(t, "google_secret", config.GoogleClientSecret)
				require.Len(t, config.Accounts, 1)
				assert.Equal(t, "account1", config.Accounts[0].Name)
				assert.Equal(t, "pop3", config.Accounts[0].Protocol)
				assert.Equal(t, "pop.example.com:995", config.Accounts[0].Server)
				assert.Equal(t, "user1", config.Accounts[0].User)
				assert.Equal(t, "pass1", config.Accounts[0].Pass)
				assert.Equal(t, "token1", config.Accounts[0].GmailRefreshToken)
			},
		},
		{
			name:          "Missing file",
			path:          func(t *testing.T) string { return "non_existent_file.json" },
			expectError:   true,
			expectedError: "",
		},
		{
			name: "Malformed JSON",
			path: func(t *testing.T) string {
				malformedConfigContent := `{"invalid_json"`
				malformedConfigFile, err := os.CreateTemp("", "malformed_config.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(malformedConfigFile.Name()) })
				_, err = malformedConfigFile.WriteString(malformedConfigContent)
				require.NoError(t, err)
				malformedConfigFile.Close()
				return malformedConfigFile.Name()
			},
			expectError: true,
		},
		{
			name: "Missing required fields",
			path: func(t *testing.T) string {
				missingFieldsConfigContent := `{
					"poll_interval_minutes": 15,
					"google_client_id": "google_id",
					"google_client_secret": "google_secret",
					"accounts": [
						{
							"name": "account1",
							"protocol": "pop3",
							"user": "user1",
							"pass": "pass1"
						}
					]
				}`
				missingFieldsConfigFile, err := os.CreateTemp("", "missing_fields_config.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(missingFieldsConfigFile.Name()) })
				_, err = missingFieldsConfigFile.WriteString(missingFieldsConfigContent)
				require.NoError(t, err)
				missingFieldsConfigFile.Close()
				return missingFieldsConfigFile.Name()
			},
			expectError:   true,
			expectedError: "server is required",
		},
		{
			name: "Invalid protocol",
			path: func(t *testing.T) string {
				invalidProtocolConfigContent := `{
					"poll_interval_minutes": 15,
					"google_client_id": "google_id",
					"google_client_secret": "google_secret",
					"accounts": [
						{
							"name": "account1",
							"protocol": "smtp",
							"server": "smtp.example.com:465",
							"user": "user1",
							"pass": "pass1",
							"gmail_refresh_token": "token1"
						}
					]
				}`
				invalidProtocolConfigFile, err := os.CreateTemp("", "invalid_protocol_config.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(invalidProtocolConfigFile.Name()) })
				_, err = invalidProtocolConfigFile.WriteString(invalidProtocolConfigContent)
				require.NoError(t, err)
				invalidProtocolConfigFile.Close()
				return invalidProtocolConfigFile.Name()
			},
			expectError:   true,
			expectedError: "protocol must be 'pop3' or 'imap'",
		},
		{
			name: "Valid OAuth2 IMAP account",
			path: func(t *testing.T) string {
				content := `{
					"poll_interval_minutes": 10,
					"google_client_id": "google_id",
					"google_client_secret": "google_secret",
					"accounts": [
						{
							"name": "Outlook Account",
							"protocol": "imap",
							"server": "outlook.office365.com:993",
							"user": "me@outlook.com",
							"auth_method": "oauth2",
							"ms_client_id": "azure-client-id",
							"ms_client_secret": "azure-client-secret",
							"ms_refresh_token": "refresh-token",
							"gmail_refresh_token": "gmail-token"
						}
					]
				}`
				f, err := os.CreateTemp("", "oauth2_imap_config.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(f.Name()) })
				_, err = f.WriteString(content)
				require.NoError(t, err)
				f.Close()
				return f.Name()
			},
			expectError: false,
			check: func(t *testing.T, config *Config) {
				require.Len(t, config.Accounts, 1)
				acc := config.Accounts[0]
				assert.Equal(t, "oauth2", acc.AuthMethod)
				assert.Equal(t, "azure-client-id", acc.MSClientID)
				assert.Equal(t, "azure-client-secret", acc.MSClientSecret)
				assert.Equal(t, "refresh-token", acc.MSRefreshToken)
				assert.Empty(t, acc.Pass)
			},
		},
		{
			name: "OAuth2 IMAP account missing ms_client_id",
			path: func(t *testing.T) string {
				content := `{
					"poll_interval_minutes": 10,
					"google_client_id": "gid",
					"google_client_secret": "gsecret",
					"accounts": [
						{
							"name": "Outlook Account",
							"protocol": "imap",
							"server": "outlook.office365.com:993",
							"user": "me@outlook.com",
							"auth_method": "oauth2",
							"ms_client_secret": "azure-client-secret",
							"ms_refresh_token": "refresh-token",
							"gmail_refresh_token": "gmail-token"
						}
					]
				}`
				f, err := os.CreateTemp("", "oauth2_missing_client_id.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(f.Name()) })
				_, err = f.WriteString(content)
				require.NoError(t, err)
				f.Close()
				return f.Name()
			},
			expectError:   true,
			expectedError: "ms_client_id is required",
		},
		{
			name: "OAuth2 IMAP account missing ms_refresh_token",
			path: func(t *testing.T) string {
				content := `{
					"poll_interval_minutes": 10,
					"google_client_id": "gid",
					"google_client_secret": "gsecret",
					"accounts": [
						{
							"name": "Outlook Account",
							"protocol": "imap",
							"server": "outlook.office365.com:993",
							"user": "me@outlook.com",
							"auth_method": "oauth2",
							"ms_client_id": "azure-client-id",
							"ms_client_secret": "azure-client-secret",
							"gmail_refresh_token": "gmail-token"
						}
					]
				}`
				f, err := os.CreateTemp("", "oauth2_missing_refresh.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(f.Name()) })
				_, err = f.WriteString(content)
				require.NoError(t, err)
				f.Close()
				return f.Name()
			},
			expectError:   true,
			expectedError: "ms_refresh_token is required",
		},
		{
			name: "Invalid auth_method value",
			path: func(t *testing.T) string {
				content := `{
					"poll_interval_minutes": 10,
					"google_client_id": "gid",
					"google_client_secret": "gsecret",
					"accounts": [
						{
							"name": "Account",
							"protocol": "imap",
							"server": "imap.example.com:993",
							"user": "user1",
							"pass": "pass1",
							"auth_method": "kerberos",
							"gmail_refresh_token": "gmail-token"
						}
					]
				}`
				f, err := os.CreateTemp("", "invalid_auth_method.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(f.Name()) })
				_, err = f.WriteString(content)
				require.NoError(t, err)
				f.Close()
				return f.Name()
			},
			expectError:   true,
			expectedError: "auth_method must be 'password' or 'oauth2'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadConfig(tc.path(t))

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
