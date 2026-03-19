package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Account struct {
	Name              string `json:"name"`
	Pop3Server        string `json:"pop3_server"`
	Pop3User          string `json:"pop3_user"`
	Pop3Pass          string `json:"pop3_pass"`
	GmailRefreshToken string `json:"gmail_refresh_token"`
}

type Config struct {
	PollIntervalMinutes int       `json:"poll_interval_minutes"`
	GoogleClientID      string    `json:"google_client_id"`
	GoogleClientSecret  string    `json:"google_client_secret"`
	Accounts            []Account `json:"accounts"`
}

func LoadConfig(path string) (*Config, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	if err := config.validate(); err != nil {
		return nil, err
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
	for _, account := range c.Accounts {
		if account.Name == "" {
			return errors.New("account name is required")
		}
		if account.Pop3Server == "" {
			return errors.New("pop3_server is required")
		}
		if account.Pop3User == "" {
			return errors.New("pop3_user is required")
		}
		if account.Pop3Pass == "" {
			return errors.New("pop3_pass is required")
		}
		if account.GmailRefreshToken == "" {
			return errors.New("gmail_refresh_token is required")
		}
	}
	return nil
}