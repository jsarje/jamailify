package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Account struct {
	Name              string `json:"name"`
	Protocol          string `json:"protocol"`
	Server            string `json:"server"`
	User              string `json:"user"`
	Pass              string `json:"pass"`
	GmailRefreshToken string `json:"gmail_refresh_token"`
	NoTls             bool   `json:"no_tls,omitempty"`
}

type Config struct {
	PollIntervalMinutes int       `json:"poll_interval_minutes"`
	GoogleClientID      string    `json:"google_client_id"`
	GoogleClientSecret  string    `json:"google_client_secret"`
	Accounts            []Account `json:"accounts"`
	// MaxMessagesToCheck limits how many newest messages will be inspected per run.
	// If zero, a sensible default is used (2000).
	MaxMessagesToCheck int `json:"max_messages_to_check"`
	// SyncWindowDays controls how far back (in days) messages will be considered for sync.
	// If zero, defaults to 7 days.
	SyncWindowDays int `json:"sync_window_days"`
}

func LoadConfig(path string) (*Config, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

func (c *Config) validate() error {
	if c.PollIntervalMinutes <= 0 {
		return errors.New("poll_interval_minutes must be greater than 0")
	}
	if c.GoogleClientID == "" {
		return errors.New("google_client_id is required")
	}
	if c.GoogleClientSecret == "" {
		return errors.New("google_client_secret is required")
	}
	if len(c.Accounts) == 0 {
		return errors.New("at least one account is required")
	}
	for i, account := range c.Accounts {
		if account.Name == "" {
			return fmt.Errorf("account %d: name is required", i)
		}
		if account.Protocol == "" {
			return fmt.Errorf("account %q: protocol is required", account.Name)
		}
		if account.Protocol != "pop3" && account.Protocol != "imap" {
			return fmt.Errorf("account %q: protocol must be 'pop3' or 'imap'", account.Name)
		}
		if account.Server == "" {
			return fmt.Errorf("account %q: server is required", account.Name)
		}
		if account.User == "" {
			return fmt.Errorf("account %q: user is required", account.Name)
		}
		if account.Pass == "" {
			return fmt.Errorf("account %q: pass is required", account.Name)
		}
		if account.GmailRefreshToken == "" {
			return fmt.Errorf("account %q: gmail_refresh_token is required", account.Name)
		}
	}
	return nil
}
